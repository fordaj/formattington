package formattington

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// The release "full" build embeds the bundled formatter executables for
// the target platform as a zip archive (embedded_full.go). go:embed cannot
// execute the embedded bytes in place, so the first lookup unpacks them
// into a per-user cache directory and the extracted tree is used from
// then on. Lite builds embed nothing and this file is a no-op for them.
//
// Each payload extracts into its own immutable "formatters-<fingerprint>"
// directory, so a binary built from a different formatter payload never
// reuses or corrupts another payload's tree, and there is no
// read-modify-replace window for a concurrent process to observe.

var (
	embeddedOnce sync.Once
	embeddedRoot string
	embeddedErr  error
)

// embeddedFingerprintFile names the marker file holding the fingerprint
// of the extracted payload at the formatters directory root.
const embeddedFingerprintFile = ".fingerprint"

// Extraction limits. They are generous relative to the real payload (the
// bundled formatters total well under 200 MiB) and exist to bound the
// damage of a hostile or corrupt archive.
var (
	maxEmbeddedMemberBytes = uint64(256 << 20) // 256 MiB per member
	maxEmbeddedTotalBytes  = uint64(1 << 30)   // 1 GiB per payload
	maxEmbeddedFiles       = 4096              // member count
)

// embeddedFormattersRoot returns the directory holding the extracted
// formatter executables, unpacking them on first use. It returns "" with
// a nil error when the binary embeds no formatters. The returned
// directory holds "<tool>/v<version>/<platform>/<executable>", the exact
// layout the bundled-tool resolution expects from a formatters/ directory.
func embeddedFormattersRoot() (string, error) {
	embeddedOnce.Do(func() {
		embeddedRoot, embeddedErr = installEmbeddedFormatters(embeddedFormattersZip, embeddedFormattersCacheDir())
	})
	return embeddedRoot, embeddedErr
}

// embeddedFormattersCacheDir is the per-user cache directory for the
// extracted bundles. It falls back to the system temporary directory with
// a per-UID suffix when the user cache directory cannot be resolved; the
// fallback is created with the same user-only permissions, or the
// extraction fails closed.
func embeddedFormattersCacheDir() string {
	base, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(base, "formattington")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("formattington-%d", os.Getuid()))
}

// secureMkdirAll creates dir (and parents) and then forces 0700 on it.
// A pre-existing directory that we own (or one we create) is tightened to
// 0700; a directory owned by someone else cannot be tightened and fails
// the extraction rather than being trusted.
func secureMkdirAll(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("cannot secure cache directory %s to 0700: %w", dir, err)
	}
	return nil
}

// installEmbeddedFormatters unpacks the payload zip (one
// "<tool>/v<version>/<platform>/<executable>" member per bundled tool)
// into a fingerprint-named directory under cacheDir and returns it. When
// the extracted tree already matches the payload, it is reused as-is. A
// payload with no regular files yields "" and a nil error. Sibling
// directories from older payloads are garbage-collected on a best-effort
// basis.
func installEmbeddedFormatters(payload []byte, cacheDir string) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	if err := secureMkdirAll(cacheDir); err != nil {
		return "", err
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", fmt.Errorf("embedded formatter archive is unreadable: %w", err)
	}
	files, err := regularZipFiles(reader.File)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", nil
	}

	fingerprint := zipFingerprint(files)
	dir := filepath.Join(cacheDir, "formatters-"+fingerprint)
	if extractedFormattersMatch(dir, fingerprint, files) {
		collectGarbage(cacheDir, filepath.Base(dir))
		return dir, nil
	}

	partial := fmt.Sprintf("%s.partial-%d", dir, os.Getpid())
	if err := os.RemoveAll(partial); err != nil {
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(partial)
		}
	}()

	var total uint64
	for _, file := range files {
		target := filepath.Join(partial, filepath.FromSlash(file.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return "", err
		}
		written, err := extractZipMember(file, target)
		if err != nil {
			return "", fmt.Errorf("unpacking embedded formatter %q: %w", file.Name, err)
		}
		total += written
		if total > maxEmbeddedTotalBytes {
			return "", fmt.Errorf("embedded formatter payload exceeds the %d byte limit", maxEmbeddedTotalBytes)
		}
	}
	if err := os.WriteFile(filepath.Join(partial, embeddedFingerprintFile), []byte(fingerprint+"\n"), 0o600); err != nil {
		return "", err
	}

	// The payload's directory is immutable: it never exists until this
	// rename makes it visible, so no process ever observes a partially
	// populated tree.
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("replacing the extracted formatters in %s: %w", dir, err)
	}
	if err := os.Rename(partial, dir); err != nil {
		// A concurrent process may have extracted the same payload; keep
		// whichever complete tree won the rename.
		_ = os.RemoveAll(partial)
		if extractedFormattersMatch(dir, fingerprint, files) {
			committed = true
			collectGarbage(cacheDir, filepath.Base(dir))
			return dir, nil
		}
		return "", err
	}
	committed = true
	collectGarbage(cacheDir, filepath.Base(dir))
	return dir, nil
}

// collectGarbage removes other "formatters-*" trees from older payloads
// and stale "*.partial-*" directories. Errors are ignored: a directory
// that cannot be removed (e.g. a running binary locks it on Windows) is
// simply left in place.
func collectGarbage(cacheDir, keep string) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			continue
		}
		if name == keep {
			continue
		}
		if !strings.HasPrefix(name, "formatters-") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(cacheDir, name))
	}
}

// regularZipFiles keeps the regular, safely-named, non-escaping members
// of the embedded archive, sorted by name for a stable fingerprint. It
// fails on duplicate cleaned member names and on members that exceed the
// extraction limits; anything else suspicious (absolute paths, drive
// letters, backslash separators, "..", non-regular files) is dropped.
func regularZipFiles(list []*zip.File) ([]*zip.File, error) {
	files := make([]*zip.File, 0, len(list))
	seen := make(map[string]bool, len(list))
	var total uint64
	for _, file := range list {
		if file.FileInfo().IsDir() {
			continue
		}
		if !file.FileInfo().Mode().IsRegular() {
			// Directories and symlinks are dropped silently; the payload
			// never carries them.
			continue
		}
		name := file.Name
		if strings.HasPrefix(name, "/") || strings.ContainsRune(name, '\\') {
			continue
		}
		// Windows drive letters (C:/...) and UNC-like prefixes are never
		// valid members of the payload.
		if len(name) >= 2 && name[1] == ':' {
			continue
		}
		if strings.HasPrefix(name, "//") {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
			continue
		}
		if file.UncompressedSize64 > maxEmbeddedMemberBytes {
			return nil, fmt.Errorf("embedded archive member %q claims to be %d bytes, over the %d byte limit",
				name, file.UncompressedSize64, maxEmbeddedMemberBytes)
		}
		total += file.UncompressedSize64
		if total > maxEmbeddedTotalBytes {
			return nil, fmt.Errorf("embedded archive payload exceeds the %d byte limit", maxEmbeddedTotalBytes)
		}
		if seen[clean] {
			return nil, fmt.Errorf("embedded archive contains duplicate member %q", clean)
		}
		seen[clean] = true
		file.Name = clean
		files = append(files, file)
		if len(files) > maxEmbeddedFiles {
			return nil, fmt.Errorf("embedded archive contains more than %d members", maxEmbeddedFiles)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

// zipFingerprint identifies the payload by member name, size, and CRC-32,
// all of which are available from the archive index without decompressing
// the members. It exists to detect a changed payload, so an upgraded
// binary never reuses a stale extraction; it is also the directory name.
func zipFingerprint(files []*zip.File) string {
	hash := sha256.New()
	for _, file := range files {
		fmt.Fprintf(hash, "%s:%d:%x\n", file.Name, file.UncompressedSize64, file.CRC32)
	}
	return hex.EncodeToString(hash.Sum(nil))[:32]
}

// extractedFormattersMatch reports whether dir already holds the marker
// for this payload and every member of the payload itself (validated by
// size, which the fingerprint covers by CRC-32 at extraction time).
func extractedFormattersMatch(dir, fingerprint string, files []*zip.File) bool {
	marker, err := os.ReadFile(filepath.Join(dir, embeddedFingerprintFile))
	if err != nil || strings.TrimSpace(string(marker)) != fingerprint {
		return false
	}
	for _, file := range files {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(file.Name)))
		if err != nil || info.IsDir() || info.Size() != int64(file.UncompressedSize64) {
			return false
		}
	}
	return true
}

// extractZipMember writes one archive member to target and marks it
// executable: bundled formatters ship as executables, and the embedded
// bytes carry no usable permission bits. It returns the number of bytes
// written; a member that yields more than maxEmbeddedMemberBytes is an
// error even if its header lied.
func extractZipMember(file *zip.File, target string) (uint64, error) {
	src, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(dst, io.LimitReader(src, int64(maxEmbeddedMemberBytes)+1))
	closeErr := dst.Close()
	if copyErr != nil {
		return 0, copyErr
	}
	if closeErr != nil {
		return 0, closeErr
	}
	if uint64(written) > maxEmbeddedMemberBytes {
		return 0, fmt.Errorf("archive member %q exceeds the %d byte limit", file.Name, maxEmbeddedMemberBytes)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return 0, err
	}
	return uint64(written), nil
}

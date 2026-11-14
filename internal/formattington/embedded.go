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
// The extraction lives in a single fixed directory with a fingerprint
// marker, so the cache holds at most one bundle; a binary built from a
// different formatter payload replaces it wholesale.

var (
	embeddedOnce sync.Once
	embeddedRoot string
	embeddedErr  error
)

// embeddedFingerprintFile names the marker file holding the fingerprint
// of the extracted payload at the formatters directory root.
const embeddedFingerprintFile = ".fingerprint"

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
// extracted bundles. It falls back to the system temporary directory when
// the user cache directory cannot be resolved.
func embeddedFormattersCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "formattington")
}

// installEmbeddedFormatters unpacks the payload zip (one
// "<tool>/v<version>/<platform>/<executable>" member per bundled tool)
// into the "formatters" directory under cacheDir and returns that
// directory. When the extracted tree already matches the payload, it is
// reused as-is. A payload with no regular files yields "" and a nil
// error.
func installEmbeddedFormatters(payload []byte, cacheDir string) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", fmt.Errorf("embedded formatter archive is unreadable: %w", err)
	}
	files := regularZipFiles(reader.File)
	if len(files) == 0 {
		return "", nil
	}

	fingerprint := zipFingerprint(files)
	dir := filepath.Join(cacheDir, "formatters")
	if extractedFormattersMatch(dir, fingerprint, files) {
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

	for _, file := range files {
		target := filepath.Join(partial, filepath.FromSlash(file.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := extractZipMember(file, target); err != nil {
			return "", fmt.Errorf("unpacking embedded formatter %q: %w", file.Name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(partial, embeddedFingerprintFile), []byte(fingerprint+"\n"), 0o644); err != nil {
		return "", err
	}

	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("replacing the extracted formatters in %s: %w", dir, err)
	}
	if err := os.Rename(partial, dir); err != nil {
		// A concurrent process may have extracted the same payload; keep
		// whichever tree is complete.
		_ = os.RemoveAll(partial)
		if extractedFormattersMatch(dir, fingerprint, files) {
			committed = true
			return dir, nil
		}
		return "", err
	}
	committed = true
	return dir, nil
}

// regularZipFiles keeps the regular, safely-named, non-escaping members of
// the embedded archive, sorted by name for a stable fingerprint.
func regularZipFiles(list []*zip.File) []*zip.File {
	files := make([]*zip.File, 0, len(list))
	for _, file := range list {
		if file.FileInfo().IsDir() {
			continue
		}
		if !file.FileInfo().Mode().IsRegular() {
			continue
		}
		name := file.Name
		if strings.HasPrefix(name, "/") || strings.ContainsRune(name, '\\') {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
			continue
		}
		file.Name = clean
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files
}

// zipFingerprint identifies the payload by member name, size, and CRC-32,
// all of which are available from the archive index without decompressing
// the members. It exists to detect a changed payload, so an upgraded
// binary never reuses a stale extraction.
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
// bytes carry no usable permission bits.
func extractZipMember(file *zip.File, target string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(target, 0o755)
}

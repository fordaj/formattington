package formattington

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// buildPayload zips entries ("name" -> contents) into an in-memory
// archive the way release.bash produces assets/formatters.zip.
func buildPayload(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	for _, name := range names {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := member.Write([]byte(entries[name])); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func TestInstallEmbeddedFormattersExtractsThePayload(t *testing.T) {
	payload := buildPayload(t, map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt":       "#!/bin/sh\necho shfmt\n",
		"black/v26.10.1/linux-x86_64/black":      "#!/bin/sh\necho black\n",
		"black/v26.10.1/linux-x86_64/README.txt": "docs\n",
	})

	cacheDir := t.TempDir()
	dir, err := installEmbeddedFormatters(payload, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters: %v", err)
	}
	want := filepath.Join(cacheDir, "formatters")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	for name, contents := range map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt":  "#!/bin/sh\necho shfmt\n",
		"black/v26.10.1/linux-x86_64/black": "#!/bin/sh\necho black\n",
	} {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != contents {
			t.Errorf("%s = %q, want %q", name, got, contents)
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
			if err != nil {
				t.Fatalf("stat %s: %v", name, err)
			}
			if info.Mode().Perm() != 0o755 {
				t.Errorf("%s mode = %v, want 0755", name, info.Mode().Perm())
			}
		}
	}
}

func TestInstallEmbeddedFormattersReusesAMatchingExtraction(t *testing.T) {
	payload := buildPayload(t, map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt": "#!/bin/sh\necho shfmt\n",
	})

	cacheDir := t.TempDir()
	dir, err := installEmbeddedFormatters(payload, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters: %v", err)
	}
	marker, err := os.ReadFile(filepath.Join(dir, embeddedFingerprintFile))
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}

	again, err := installEmbeddedFormatters(payload, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters (second call): %v", err)
	}
	if again != dir {
		t.Errorf("second call returned %q, want %q", again, dir)
	}
	after, err := os.ReadFile(filepath.Join(dir, embeddedFingerprintFile))
	if err != nil {
		t.Fatalf("read marker after reuse: %v", err)
	}
	if !bytes.Equal(marker, after) {
		t.Error("marker changed between calls; the extraction was not reused")
	}
}

func TestInstallEmbeddedFormattersReExtractsAfterCorruption(t *testing.T) {
	payload := buildPayload(t, map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt": "#!/bin/sh\necho shfmt\n",
	})

	cacheDir := t.TempDir()
	dir, err := installEmbeddedFormatters(payload, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters: %v", err)
	}
	victim := filepath.Join(dir, "shfmt", "v3.14.1", "linux-x86_64", "shfmt")
	if err := os.WriteFile(victim, []byte("corrupt"), 0o755); err != nil {
		t.Fatalf("corrupt the extraction: %v", err)
	}

	dir, err = installEmbeddedFormatters(payload, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters (after corruption): %v", err)
	}
	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(got) != "#!/bin/sh\necho shfmt\n" {
		t.Errorf("restored contents = %q, want the payload contents", got)
	}
}

func TestInstallEmbeddedFormattersRefreshesAStalePayload(t *testing.T) {
	cacheDir := t.TempDir()
	first := buildPayload(t, map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt": "#!/bin/sh\necho v1\n",
	})
	dir, err := installEmbeddedFormatters(first, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters: %v", err)
	}

	updated := buildPayload(t, map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt": "#!/bin/sh\necho v2-longer\n",
	})
	dir, err = installEmbeddedFormatters(updated, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters (updated payload): %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "shfmt", "v3.14.1", "linux-x86_64", "shfmt"))
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(got) != "#!/bin/sh\necho v2-longer\n" {
		t.Errorf("contents = %q, want the updated payload", got)
	}
}

func TestInstallEmbeddedFormattersRejectsZipSlip(t *testing.T) {
	payload := buildPayload(t, map[string]string{
		"../escape.txt":                    "escape\n",
		"shfmt/v3.14.1/linux-x86_64/shfmt": "#!/bin/sh\necho shfmt\n",
	})
	cacheDir := t.TempDir()
	dir, err := installEmbeddedFormatters(payload, cacheDir)
	if err != nil {
		t.Fatalf("installEmbeddedFormatters: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "escape.txt")); !os.IsNotExist(err) {
		t.Error("a member escaped the extraction directory (zip slip)")
	}
	if _, err := os.Stat(filepath.Join(dir, "shfmt", "v3.14.1", "linux-x86_64", "shfmt")); err != nil {
		t.Errorf("the safe member must still be extracted: %v", err)
	}
}

func TestInstallEmbeddedFormattersIgnoresDirectoriesAndGarbage(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if _, err := writer.Create("shfmt/"); err != nil {
		t.Fatalf("create dir member: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if dir, err := installEmbeddedFormatters(buffer.Bytes(), t.TempDir()); err != nil || dir != "" {
		t.Errorf("installEmbeddedFormatters = %q, %v; want \"\", nil", dir, err)
	}
}

func TestInstallEmbeddedFormattersEmptyPayload(t *testing.T) {
	for _, payload := range [][]byte{nil, {}} {
		if dir, err := installEmbeddedFormatters(payload, t.TempDir()); err != nil || dir != "" {
			t.Errorf("installEmbeddedFormatters = %q, %v; want \"\", nil", dir, err)
		}
	}
}

func TestInstallEmbeddedFormattersRejectsACorruptArchive(t *testing.T) {
	if dir, err := installEmbeddedFormatters([]byte("not a zip"), t.TempDir()); err == nil || dir != "" {
		t.Errorf("installEmbeddedFormatters = %q, %v; want an error", dir, err)
	}
}

// TestFormattersDirPrefersTheEmbeddedBundle resets the extraction state so
// the embedded path is exercised, and restores it afterwards. It cannot
// run alongside other tests that resolve formatters, so it serializes on
// the package's test order instead of using t.Parallel.
func TestFormattersDirPrefersTheEmbeddedBundle(t *testing.T) {
	payload := buildPayload(t, map[string]string{
		"shfmt/v3.14.1/linux-x86_64/shfmt": "#!/bin/sh\necho shfmt\n",
	})

	saveRoot, saveErr, saveZip := embeddedRoot, embeddedErr, embeddedFormattersZip
	t.Cleanup(func() {
		embeddedOnce, embeddedRoot, embeddedErr = sync.Once{}, saveRoot, saveErr
		embeddedFormattersZip = saveZip
	})

	cacheDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", "")
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("HOME", cacheDir)
	t.Setenv("LOCALAPPDATA", cacheDir)

	embeddedOnce = sync.Once{}
	embeddedRoot, embeddedErr = "", nil
	embeddedFormattersZip = payload

	got := FormattersDir()
	want := filepath.Join(embeddedFormattersCacheDir(), "formatters")
	if got != want {
		t.Errorf("FormattersDir = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(got, "shfmt", "v3.14.1", "linux-x86_64", "shfmt")); err != nil {
		t.Errorf("embedded formatter was not extracted: %v", err)
	}
}

func TestFormattersDirFallsBackWhenTheBundleFailsToUnpack(t *testing.T) {
	saveRoot, saveErr, saveZip := embeddedRoot, embeddedErr, embeddedFormattersZip
	t.Cleanup(func() {
		embeddedOnce, embeddedRoot, embeddedErr = sync.Once{}, saveRoot, saveErr
		embeddedFormattersZip = saveZip
	})

	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", "")

	embeddedOnce = sync.Once{}
	embeddedRoot, embeddedErr = "", nil
	embeddedFormattersZip = []byte("not a zip")

	got := FormattersDir()
	if strings.HasPrefix(got, filepath.Join(os.TempDir(), "formattington")) {
		t.Fatalf("FormattersDir = %q, want a repository fallback", got)
	}
}

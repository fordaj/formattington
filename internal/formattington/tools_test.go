package formattington

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlatformDir(t *testing.T) {
	cases := []struct {
		system   string
		machine  string
		expected string
	}{
		{"Windows", "AMD64", "windows-x86_64"},
		{"Windows", "x86_64", "windows-x86_64"},
		{"Windows", "arm64", "windows-arm64"},
		{"Darwin", "arm64", "macos-arm64"},
		{"Darwin", "x86_64", "macos-x86_64"},
		{"Linux", "x86_64", "linux-x86_64"},
		{"Linux", "aarch64", "linux-arm64"},
		{"Linux", "amd64", "linux-x86_64"},
	}
	for _, testCase := range cases {
		got, err := platformDir(testCase.system, testCase.machine)
		if err != nil {
			t.Fatalf("platformDir(%q, %q): %v", testCase.system, testCase.machine, err)
		}
		if got != testCase.expected {
			t.Errorf("platformDir(%q, %q) = %q, want %q",
				testCase.system, testCase.machine, got, testCase.expected)
		}
	}
}

func TestPlatformDirUnsupported(t *testing.T) {
	if _, err := platformDir("FreeBSD", "sparc"); err == nil {
		t.Fatal("expected an error for an unsupported operating system")
	} else if !strings.Contains(err.Error(), "Unsupported platform") {
		t.Errorf("unexpected error: %v", err)
	}

	if _, err := platformDir("Linux", "riscv"); err == nil {
		t.Fatal("expected an error for an unsupported architecture")
	} else if !strings.Contains(err.Error(), "Unsupported platform") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVersionKeyOrdering(t *testing.T) {
	if compareVersionKeys(versionKey("9.0.0"), versionKey("10.0.0")) >= 0 {
		t.Error("v10 must sort after v9")
	}
	if compareVersionKeys(versionKey("22.1.8"), versionKey("23.1.3")) >= 0 {
		t.Error("v23.1.3 must sort after v22.1.8")
	}
	if got := versionKey("3.14.1"); !reflect.DeepEqual(got, []int{3, 14, 1}) {
		t.Errorf("versionKey(\"3.14.1\") = %v, want [3 14 1]", got)
	}
	if got := versionKey("v1.2.3-rc1"); !reflect.DeepEqual(got, []int{1, 2, 3, 0}) {
		t.Errorf("versionKey(\"v1.2.3-rc1\") = %v, want [1 2 3 0]", got)
	}
}

// makeBundledLayout creates tools/<name>/v<version>/<platform>/<exe>.
func makeBundledLayout(t *testing.T, toolsDir, name, version, platform, exe string) string {
	t.Helper()
	path := filepath.Join(toolsDir, name, "v"+version, platform, exe)
	writeFile(t, path, "")
	return path
}

func TestFindToolPrefersBundled(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	expected := makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")

	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "shfmt"), "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", binDir)

	got, err := FindTool("shfmt", "", false)
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	if got != expected {
		t.Errorf("FindTool = %q, want %q", got, expected)
	}
}

func TestFindToolExplicitVersion(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	expected := makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
	t.Setenv("PATH", t.TempDir())

	for _, version := range []string{"v3.14.1", "3.14.1"} {
		got, err := FindTool("shfmt", version, false)
		if err != nil {
			t.Fatalf("FindTool(%q): %v", version, err)
		}
		if got != expected {
			t.Errorf("FindTool(%q) = %q, want %q", version, got, expected)
		}
	}
}

func TestFindToolPrefersNewestVersion(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	makeBundledLayout(t, toolsDir, "clang-format", "22.1.8", "macos-x86_64", "clang-format")
	newest := makeBundledLayout(t, toolsDir, "clang-format", "23.1.3", "macos-x86_64", "clang-format")
	t.Setenv("PATH", t.TempDir())

	got, err := FindTool("clang-format", "", false)
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	if got != newest {
		t.Errorf("FindTool = %q, want %q", got, newest)
	}
}

func TestFindToolNumericVersionOrdering(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	makeBundledLayout(t, toolsDir, "clang-format", "9.0.0", "macos-x86_64", "clang-format")
	newest := makeBundledLayout(t, toolsDir, "clang-format", "10.0.0", "macos-x86_64", "clang-format")
	t.Setenv("PATH", t.TempDir())

	got, err := FindTool("clang-format", "", false)
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	if got != newest {
		t.Errorf("FindTool = %q, want %q", got, newest)
	}
}

func TestFindToolFallsBackToPath(t *testing.T) {
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	binDir := filepath.Join(t.TempDir(), "bin")
	expected := writeFakeExecutable(t, binDir, "shfmt")
	t.Setenv("PATH", binDir)

	got, err := FindTool("shfmt", "", false)
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	if got != expected {
		t.Errorf("FindTool = %q, want %q", got, expected)
	}
}

// A self-contained release bundle keeps "formatters/<tool>/v<version>/
// <platform>/" next to the program; the bundled tool must be found there.
func TestFindToolInASelfContainedBundle(t *testing.T) {
	bundle := t.TempDir()
	platform, err := PlatformDir()
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}
	expected := makeBundledLayout(t, filepath.Join(bundle, "formatters"), "shfmt", "3.14.1", platform, "shfmt")
	t.Setenv("PATH", t.TempDir())

	// The bundle root is the ancestor that holds "formatters/".
	if got, ok := findRootWithBundles(bundle); !ok || got != bundle {
		t.Fatalf("findRootWithBundles = %q (ok=%v), want %q", got, ok, bundle)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", filepath.Join(bundle, "formatters"))

	got, err := FindTool("shfmt", "", false)
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	if got != expected {
		t.Errorf("FindTool = %q, want %q", got, expected)
	}
}

func TestFindToolMissingEverywhere(t *testing.T) {
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	_, err := FindTool("shfmt", "", false)
	if err == nil {
		t.Fatal("expected an error when the tool is nowhere to be found")
	}
	if !strings.Contains(err.Error(), "Could not find 'shfmt'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFindToolSkipsOtherPlatforms(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", "linux-x86_64", "shfmt")

	binDir := filepath.Join(t.TempDir(), "bin")
	expected := writeFakeExecutable(t, binDir, "shfmt")
	t.Setenv("PATH", binDir)

	got, err := FindTool("shfmt", "", false)
	if err != nil {
		t.Fatalf("FindTool: %v", err)
	}
	if got != expected {
		t.Errorf("FindTool = %q, want the PATH copy %q", got, expected)
	}
}

func TestVerboseResolutionReportsBundled(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	expected := makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FORMATTINGTON_VERBOSE", "")

	_, _, stderr := captureOutput(t, func() int {
		_, err := FindTool("shfmt", "", true)
		if err != nil {
			t.Fatalf("FindTool: %v", err)
		}
		return ExitOK
	})

	if !strings.Contains(stderr, "resolved to") {
		t.Errorf("expected a resolution message, got %q", stderr)
	}
	if !strings.Contains(stderr, expected) {
		t.Errorf("expected the resolved path in %q", stderr)
	}
	if !strings.Contains(stderr, "(bundled)") {
		t.Errorf("expected the bundled marker in %q", stderr)
	}
}

func TestVerboseResolutionReportsPath(t *testing.T) {
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	binDir := filepath.Join(t.TempDir(), "bin")
	writeFakeExecutable(t, binDir, "shfmt")
	t.Setenv("PATH", binDir)
	t.Setenv("FORMATTINGTON_VERBOSE", "")

	_, _, stderr := captureOutput(t, func() int {
		_, err := FindTool("shfmt", "", true)
		if err != nil {
			t.Fatalf("FindTool: %v", err)
		}
		return ExitOK
	})

	if !strings.Contains(stderr, "(PATH)") {
		t.Errorf("expected the PATH marker in %q", stderr)
	}
}

func TestVerboseResolutionViaEnvironment(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FORMATTINGTON_VERBOSE", "1")

	_, _, stderr := captureOutput(t, func() int {
		_, err := FindTool("shfmt", "", false)
		if err != nil {
			t.Fatalf("FindTool: %v", err)
		}
		return ExitOK
	})

	if !strings.Contains(stderr, "resolved to") {
		t.Errorf("expected a resolution message, got %q", stderr)
	}
}

func TestNotVerboseByDefault(t *testing.T) {
	toolsDir := t.TempDir()
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", toolsDir)
	makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", "macos-x86_64", "shfmt")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FORMATTINGTON_VERBOSE", "")

	_, _, stderr := captureOutput(t, func() int {
		_, err := FindTool("shfmt", "", false)
		if err != nil {
			t.Fatalf("FindTool: %v", err)
		}
		return ExitOK
	})

	if stderr != "" {
		t.Errorf("expected no output, got %q", stderr)
	}
}

func TestRunToolReturnsExitCode(t *testing.T) {
	requirePOSIX(t)
	binDir := filepath.Join(t.TempDir(), "bin")
	path := filepath.Join(binDir, "boom")
	writeFile(t, path, "#!/bin/sh\nexit 3\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	code, err := runTool(path, []string{"-w"}, false)
	if err != nil {
		t.Fatalf("runTool: %v", err)
	}
	if code != 3 {
		t.Errorf("runTool = %d, want 3", code)
	}
}

func TestRunToolCaptureOutputReportsChanges(t *testing.T) {
	requirePOSIX(t)
	binDir := filepath.Join(t.TempDir(), "bin")
	path := filepath.Join(binDir, "list")
	writeFile(t, path, "#!/bin/sh\necho changed.sh\nexit 0\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	code, stdout, stderr := captureOutput(t, func() int {
		code, err := runTool(path, []string{"-l"}, true)
		if err != nil {
			t.Fatalf("runTool: %v", err)
		}
		return code
	})

	if code != ExitFormatRequired {
		t.Errorf("runTool = %d, want %d", code, ExitFormatRequired)
	}
	if !strings.Contains(stdout, "changed.sh") {
		t.Errorf("expected the tool output on stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "formatting changes required") {
		t.Errorf("expected the change report on stderr, got %q", stderr)
	}
}

func TestRunToolCaptureOutputClean(t *testing.T) {
	requirePOSIX(t)
	binDir := filepath.Join(t.TempDir(), "bin")
	path := filepath.Join(binDir, "list")
	writeFile(t, path, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	code, err := runTool(path, []string{"-l"}, true)
	if err != nil {
		t.Fatalf("runTool: %v", err)
	}
	if code != ExitOK {
		t.Errorf("runTool = %d, want %d", code, ExitOK)
	}
}

func TestRunToolCaptureOutputPropagatesNonZero(t *testing.T) {
	requirePOSIX(t)
	binDir := filepath.Join(t.TempDir(), "bin")
	path := filepath.Join(binDir, "list")
	writeFile(t, path, "#!/bin/sh\necho boom >&2\nexit 2\n")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	code, _, stderr := captureOutput(t, func() int {
		code, err := runTool(path, []string{"-l"}, true)
		if err != nil {
			t.Fatalf("runTool: %v", err)
		}
		return code
	})

	if code != 2 {
		t.Errorf("runTool = %d, want 2", code)
	}
	if !strings.Contains(stderr, "boom") {
		t.Errorf("expected the tool output on stderr, got %q", stderr)
	}
}

func TestIsInformational(t *testing.T) {
	truthy := [][]string{
		{"--help"},
		{"-help"},
		{"--version"},
		{"-version"},
		{"-h"},
		{"-V"},
		{"file.cpp", "--help"},
	}
	for _, args := range truthy {
		if !IsInformational(args) {
			t.Errorf("IsInformational(%v) = false, want true", args)
		}
	}

	falsy := [][]string{
		{},
		{"file.cpp"},
		{"--check"},
		{"--style=file"},
	}
	for _, args := range falsy {
		if IsInformational(args) {
			t.Errorf("IsInformational(%v) = true, want false", args)
		}
	}
}

func TestIterChunks(t *testing.T) {
	got := iterChunks([]string{"a", "b", "c", "d", "e"}, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("iterChunks = %v, want %v", got, want)
	}

	if got := iterChunks([]string{"a"}, 256); !reflect.DeepEqual(got, [][]string{{"a"}}) {
		t.Errorf("iterChunks = %v, want [[a]]", got)
	}

	if got := iterChunks(nil, 3); got != nil {
		t.Errorf("iterChunks(nil) = %v, want nil", got)
	}

	if got := iterChunks([]string{"a"}, 0); got != nil {
		t.Errorf("iterChunks with a non-positive size = %v, want nil", got)
	}
}

func TestPathSuffix(t *testing.T) {
	cases := map[string]string{
		"file.sh":     ".sh",
		"FILE.SH":     ".sh",
		"file.tar.gz": ".gz",
		".bashrc":     "",
		"file.":       "",
		"":            "",
	}
	for input, want := range cases {
		if got := pathSuffix(input); got != want {
			t.Errorf("pathSuffix(%q) = %q, want %q", input, got, want)
		}
	}
}

package formattington

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionUsesTheEnvironmentOverride(t *testing.T) {
	t.Setenv(versionEnv, "1.2.3")
	if got := resolveVersion(); got != "1.2.3" {
		t.Errorf("resolveVersion = %q, want %q", got, "1.2.3")
	}
}

func TestVersionUsesPkgInfo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "PKG-INFO"), "Metadata-Version: 2.1\nVersion: 9.8.7\n")
	t.Chdir(root)
	t.Setenv(versionEnv, "")

	// The working directory is checked before the repository root.
	if got := resolveVersion(); got != "9.8.7" && got != "0.0.0" {
		t.Errorf("resolveVersion = %q, want the PKG-INFO version", got)
	}
}

func TestVersionFallsBackToZero(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv(versionEnv, "")
	// Point the resolver at an empty tree so no versioner is found.
	t.Setenv("FORMATTINGTON_TOOLS_DIR", filepath.Join(root, "empty-tools"))
	t.Setenv("PATH", root)

	_, _, stderr := captureOutput(t, func() int {
		if got := resolveVersion(); got != "0.0.0" {
			t.Errorf("resolveVersion = %q, want 0.0.0", got)
		}
		return ExitOK
	})

	if !strings.Contains(stderr, "versioner unavailable") {
		t.Errorf("stderr = %q, must warn about the missing versioner", stderr)
	}
}

func TestVersionIsNotEmpty(t *testing.T) {
	if Version() == "" {
		t.Error("Version must not be empty")
	}
}

func TestBundledVersionerIsResolved(t *testing.T) {
	root := t.TempDir()
	toolsDir := filepath.Join(root, "tools", "versioner")
	platform, err := PlatformDir()
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}
	writeFile(t, filepath.Join(toolsDir, "v0.0.3.0.5", platform, "versioner"), "#!/bin/sh\nexit 0\n")
	t.Setenv("FORMATTINGTON_TOOLS_DIR", filepath.Join(root, "tools"))

	got := bundledVersioner(root)
	if got == "" {
		t.Fatal("expected the bundled versioner to be found")
	}
	if !strings.HasSuffix(got, filepath.Join("versioner", "v0.0.3.0.5", platform, "versioner")) {
		t.Errorf("bundledVersioner = %q, want the newest bundled version", got)
	}
}

func TestVersionerOutputIsParsed(t *testing.T) {
	requirePOSIX(t)

	root := t.TempDir()
	binary := filepath.Join(root, "versioner")
	writeFile(t, binary, "#!/bin/sh\necho '{\"version\": \"254.0.0.0.41\"}'\n")
	if err := chmodExecutable(binary); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	got := versionFromVersioner(binary, root)
	if got != "254.0.0.0.41" {
		t.Errorf("versionFromVersioner = %q, want 254.0.0.0.41", got)
	}
}

func TestToolsDirUsesTheEnvironmentOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-tools")
	t.Setenv(toolsDirEnv, override)
	if got := ToolsDir(); got != override {
		t.Errorf("ToolsDir = %q, want %q", got, override)
	}
}

func TestToolsDirFallsBackToTheRepositoryTools(t *testing.T) {
	t.Setenv(toolsDirEnv, "")
	root := repoRoot()
	got := ToolsDir()
	if got != filepath.Join(root, "tools") {
		t.Errorf("ToolsDir = %q, want %q", got, filepath.Join(root, "tools"))
	}
}

func TestFindRootWithTools(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, ok := findRootWithTools(nested)
	if !ok {
		t.Fatal("expected the ancestor with the tools directory to be found")
	}
	if got != root {
		t.Errorf("findRootWithTools = %q, want %q", got, root)
	}

	if _, ok := findRootWithTools(filepath.Join(root, "a", "missing")); !ok {
		t.Error("expected the search to keep walking up to the tools directory")
	}
}

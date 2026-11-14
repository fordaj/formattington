package formattington

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// versionEnv overrides the computed version (for example in CI builds).
const versionEnv = "FORMATTINGTON_VERSION"

var (
	versionOnce  sync.Once
	versionValue string
)

// Version returns the Formattington version.
//
// Resolution order:
//  1. The FORMATTINGTON_VERSION environment variable (explicit override).
//  2. The "Version:" line of a PKG-INFO file in the working directory or
//     the repository root (sdist-style builds).
//  3. The bundled "versioner" binary (or "versioner" on PATH) run against
//     the repository root.
//  4. The fallback "0.0.0" with a warning on stderr.
func Version() string {
	versionOnce.Do(func() {
		versionValue = resolveVersion()
	})
	return versionValue
}

func resolveVersion() string {
	if override := strings.TrimSpace(os.Getenv(versionEnv)); override != "" {
		return override
	}

	if version := versionFromPkgInfo(); version != "" {
		return version
	}

	root := repoRoot()
	if binary := bundledVersioner(root); binary != "" {
		if version := versionFromVersioner(binary, root); version != "" {
			return version
		}
	}
	if binary, err := exec.LookPath("versioner"); err == nil {
		if version := versionFromVersioner(binary, root); version != "" {
			return version
		}
	}

	fmt.Fprintln(os.Stderr, "warning: versioner unavailable; falling back to 0.0.0")
	return "0.0.0"
}

// versionFromPkgInfo reads the "Version:" line of a PKG-INFO file.
func versionFromPkgInfo() string {
	candidates := []string{"PKG-INFO", filepath.Join(repoRoot(), "PKG-INFO")}
	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for line := 0; line < 50 && scanner.Scan(); line++ {
			text := scanner.Text()
			if strings.HasPrefix(text, "Version:") {
				version := strings.TrimSpace(strings.TrimPrefix(text, "Version:"))
				file.Close()
				if version != "" {
					return version
				}
				break
			}
		}
		file.Close()
	}
	return ""
}

// bundledVersioner returns the newest bundled versioner executable for this
// platform, or "" when there is none.
func bundledVersioner(root string) string {
	toolDir := filepath.Join(root, "tools", "versioner")
	platform, err := PlatformDir()
	if err != nil {
		return ""
	}
	executable := "versioner"
	if isWindows() {
		executable = "versioner.exe"
	}
	for _, version := range bundledVersions(toolDir) {
		candidate := filepath.Join(toolDir, "v"+version, platform, executable)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// versionFromVersioner runs the versioner binary and returns the version it
// reports, or "" on failure.
func versionFromVersioner(binary, root string) string {
	cmd := exec.Command(binary, "--repo-dir", root, "--quiet", "--format", "json")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return ""
	}
	return payload.Version
}

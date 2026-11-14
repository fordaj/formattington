package formattington

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Exit codes used by Formattington itself. Exit codes produced by the
// underlying formatters are preserved verbatim.
const (
	// ExitOK means success: formatting was applied or nothing needed it.
	ExitOK = 0
	// ExitFormatRequired means formatting changes are required (check mode).
	ExitFormatRequired = 1
	// ExitUsage means a usage error (bad selector, bad flags, bad config).
	ExitUsage = 2
	// ExitNotFound means the formatter executable could not be found.
	ExitNotFound = 127
)

// ignoredDirNames are directory names discovery never descends into.
// Besides version-control metadata, these are the usual places generated,
// vendored, minified, or dependency files live. Hidden directories
// (starting with ".") are also skipped, along with the bundled-tools
// directory, which holds pinned binaries rather than user source.
var ignoredDirNames = map[string]bool{
	"__pycache__":  true,
	"node_modules": true,
	"vendor":       true,
	"venv":         true,
	"dist":         true,
	"build":        true,
	"tools":        true,
}

// informationalArgs are bare help/version tokens that pass through to the
// underlying tool without Formattington adding formatting flags.
var informationalArgs = map[string]bool{
	"--help":    true,
	"-help":     true,
	"--version": true,
	"-version":  true,
	"-h":        true,
	"-V":        true,
}

// IsInformational reports whether args are a bare help/version passthrough
// (no formatting flags).
func IsInformational(args []string) bool {
	for _, arg := range args {
		if informationalArgs[arg] {
			return true
		}
	}
	return false
}

// platformDir maps an OS/architecture pair onto the hyphenated
// "<os>-<arch>" name used for bundled tool directories.
func platformDir(system, machine string) (string, error) {
	systems := map[string]string{
		"windows": "windows",
		"darwin":  "macos",
		"linux":   "linux",
	}
	architectures := map[string]string{
		"amd64":   "x86_64",
		"x86_64":  "x86_64",
		"arm64":   "arm64",
		"aarch64": "arm64",
	}

	osName, ok := systems[strings.ToLower(system)]
	if !ok {
		return "", fmt.Errorf("Unsupported platform: %s-%s", system, machine)
	}
	arch, ok := architectures[strings.ToLower(machine)]
	if !ok {
		return "", fmt.Errorf("Unsupported platform: %s-%s", system, machine)
	}
	return osName + "-" + arch, nil
}

// PlatformDir returns the bundled-tools platform directory for the host.
func PlatformDir() (string, error) {
	return platformDir(runtime.GOOS, runtime.GOARCH)
}

// isWindows reports whether the host is Windows.
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// toolsDirEnv overrides the bundled-tools directory.
const toolsDirEnv = "FORMATTINGTON_TOOLS_DIR"

// ToolsDir returns the directory holding bundled tool executables.
//
// Resolution order: the FORMATTINGTON_TOOLS_DIR environment variable, then
// the "tools" directory of the nearest ancestor of the running executable
// or of the current directory.
func ToolsDir() string {
	if override := strings.TrimSpace(os.Getenv(toolsDirEnv)); override != "" {
		return override
	}
	return filepath.Join(repoRoot(), "tools")
}

// repoRoot returns the nearest ancestor directory (starting from the
// running executable, then the working directory) that contains a "tools"
// directory. It falls back to the working directory.
func repoRoot() string {
	if exe, err := os.Executable(); err == nil {
		if root, ok := findRootWithTools(filepath.Dir(exe)); ok {
			return root
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if root, ok := findRootWithTools(wd); ok {
			return root
		}
	}
	return "."
}

func findRootWithTools(start string) (string, bool) {
	dir := start
	for {
		if info, err := os.Stat(filepath.Join(dir, "tools")); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// verboseEnv reports whether FORMATTINGTON_VERBOSE enables verbose
// resolution hints.
func verboseEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FORMATTINGTON_VERBOSE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// versionKey parses a version string into a comparable numeric tuple.
// Sorting lexicographically fails at v9 vs v10; compare numeric
// components instead so the newest bundle wins. Non-numeric suffixes stop
// the digit scan; a component with no digits counts as 0.
func versionKey(version string) []int {
	text := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(version), "v"))
	parts := strings.Split(strings.ReplaceAll(text, "-", "."), ".")
	key := make([]int, 0, len(parts))
	for _, part := range parts {
		digits := ""
		for _, char := range part {
			if char >= '0' && char <= '9' {
				digits += string(char)
				continue
			}
			break
		}
		value, err := strconv.Atoi(digits)
		if err != nil {
			value = 0
		}
		key = append(key, value)
	}
	return key
}

// compareVersionKeys orders two numeric version tuples.
func compareVersionKeys(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// bundledVersions returns the version labels (without the "v" prefix) found
// under toolDir, newest first.
func bundledVersions(toolDir string) []string {
	entries, err := os.ReadDir(toolDir)
	if err != nil {
		return nil
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		versions = append(versions, strings.TrimPrefix(entry.Name(), "v"))
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return compareVersionKeys(versionKey(versions[i]), versionKey(versions[j])) > 0
	})
	return versions
}

// findBundledExecutable returns the bundled executable path, or "" when no
// matching bundle exists. A non-nil error means the platform itself is
// unsupported.
func findBundledExecutable(name, version, executable string) (string, error) {
	toolDir := filepath.Join(ToolsDir(), name)

	var versions []string
	if version != "" {
		versions = []string{strings.TrimPrefix(version, "v")}
	} else {
		versions = bundledVersions(toolDir)
	}

	platform, err := PlatformDir()
	if err != nil {
		return "", err
	}

	for _, candidate := range versions {
		bundled := filepath.Join(toolDir, "v"+candidate, platform, executable)
		if info, statErr := os.Stat(bundled); statErr == nil && !info.IsDir() {
			return bundled, nil
		}
	}
	return "", nil
}

// findOnPath returns the PATH-resolved tool (including the ".exe" fallback
// on Windows), or "" when it is not installed.
func findOnPath(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	if isWindows() {
		if path, err := exec.LookPath(name + ".exe"); err == nil {
			return path
		}
	}
	return ""
}

// FindTool resolves a bundled formatter, falling back to PATH.
//
// Bundled executables live under
// "tools/<name>/v<version>/<platform>/<executable>" where <platform> is the
// hyphenated PlatformDir value. When verbose is enabled explicitly or via
// FORMATTINGTON_VERBOSE, the resolved executable is printed to stderr.
func FindTool(name, version string, verbose bool) (string, error) {
	showVerbose := verbose || verboseEnv()
	executable := name
	if isWindows() {
		executable = name + ".exe"
	}

	bundled, err := findBundledExecutable(name, version, executable)
	if err != nil {
		return "", err
	}
	if bundled != "" {
		if showVerbose {
			fmt.Fprintf(os.Stderr, "formattington: %s resolved to %s (bundled)\n", name, bundled)
		}
		return bundled, nil
	}

	systemTool := findOnPath(name)
	if systemTool != "" {
		if showVerbose {
			fmt.Fprintf(os.Stderr, "formattington: %s resolved to %s (PATH)\n", name, systemTool)
		}
		return systemTool, nil
	}

	platform, platformErr := PlatformDir()
	if platformErr != nil {
		return "", platformErr
	}
	expected := filepath.Join(ToolsDir(), name, "<version>", platform, executable)
	return "", fmt.Errorf(
		"Could not find '%s'. Expected a bundled executable under %s or an installation on PATH.",
		name, expected,
	)
}

// runTool runs a formatter and returns its exit code.
//
// When captureOutput is set, stdout/stderr are captured and printed, and
// non-empty stdout is reported as ExitFormatRequired (formatting changes
// required). A non-nil error means the executable could not be started.
func runTool(executable string, arguments []string, captureOutput bool) (int, error) {
	cmd := exec.Command(executable, arguments...)

	if !captureOutput {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil && !isExitError(err) {
			return ExitNotFound, fmt.Errorf("could not run %s: %w", executable, err)
		}
		return cmd.ProcessState.ExitCode(), nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && !isExitError(err) {
		return ExitNotFound, fmt.Errorf("could not run %s: %w", executable, err)
	}

	code := cmd.ProcessState.ExitCode()
	if stdout.Len() > 0 {
		os.Stdout.Write(stdout.Bytes())
	}
	if stderr.Len() > 0 {
		os.Stderr.Write(stderr.Bytes())
	}
	if code != ExitOK {
		return code, nil
	}
	if strings.TrimSpace(stdout.String()) != "" {
		fmt.Fprintln(os.Stderr, "formattington: formatting changes required.")
		return ExitFormatRequired, nil
	}
	return ExitOK, nil
}

// isExitError reports whether err is the exit status of a finished process
// rather than a failure to start one.
func isExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

// pathSuffix returns the lower-cased extension of a path, mirroring
// pathlib's suffix rules: a leading dot marks a hidden file, not an
// extension, and a trailing dot marks no extension at all.
func pathSuffix(path string) string {
	name := filepath.Base(path)
	index := strings.LastIndex(name, ".")
	if index > 0 && index < len(name)-1 {
		return strings.ToLower(name[index:])
	}
	return ""
}

// iterChunks splits items into consecutive chunks of at most size entries.
// It is used to respect operating-system command-line limits when passing
// many discovered files to a single formatter invocation.
func iterChunks(items []string, size int) [][]string {
	if size <= 0 {
		return nil
	}
	var chunks [][]string
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

// contains reports whether values holds target.
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

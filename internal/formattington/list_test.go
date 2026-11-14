package formattington

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseListRows splits the piped table into a per-tool [bundled, PATH]
// map, skipping the header and separator rows.
func parseListRows(t *testing.T, stdout string) map[string][2]string {
	t.Helper()
	rows := map[string][2]string{}
	for _, line := range strings.Split(stdout, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		inner := make([]string, 0, len(cells))
		for _, cell := range cells {
			inner = append(inner, strings.TrimSpace(cell))
		}
		// A well-formed row is "| a | b | c |"; the empty cells at the
		// ends disappear with trimming of the surrounding bars.
		inner = trimEmptyCells(inner)
		if len(inner) != 3 {
			continue
		}
		if inner[0] == "Tool" || strings.Trim(inner[0], "-") == "" {
			continue
		}
		rows[inner[0]] = [2]string{inner[1], inner[2]}
	}
	return rows
}

// trimEmptyCells drops the empty fragments at both ends of a split row.
func trimEmptyCells(cells []string) []string {
	for len(cells) > 0 && cells[0] == "" {
		cells = cells[1:]
	}
	for len(cells) > 0 && cells[len(cells)-1] == "" {
		cells = cells[:len(cells)-1]
	}
	return cells
}

// writeVersionFake installs a fake executable that prints output and
// exits 0.
func writeVersionFake(t *testing.T, dir, name, output string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' %q\nexit 0\n", output)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// emptyToolsDir points FORMATTINGTON_FORMATTERS_DIR at an empty directory
// and PATH at an empty directory, so nothing resolves.
func emptyToolsDir(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "formatters"), 0o755); err != nil {
		t.Fatalf("mkdir formatters: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", filepath.Join(root, "formatters"))
	t.Setenv("FORMATTINGTON_VERBOSE", "")
	t.Setenv("PATH", filepath.Join(root, "bin"))
}

func TestListRequiresNoArguments(t *testing.T) {
	for _, argv := range [][]string{
		{"list", "extra"},
		{"list", "--series"},
		{"list", "--config", "x.json"},
		{"--check", "list"},
		{"--dry-run", "list"},
		{"--series", "list"},
		{"--config", "x.json", "list"},
	} {
		code, _, stderr := runCLI(t, argv...)
		if code != ExitUsage {
			t.Errorf("%v: exit = %d, want %d", argv, code, ExitUsage)
		}
		if !strings.Contains(stderr, "list") && !strings.Contains(stderr, "--config") {
			t.Errorf("%v: stderr = %q, must explain the problem", argv, stderr)
		}
	}
}

func TestListWithNoToolsReportsDashes(t *testing.T) {
	emptyToolsDir(t)

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	for _, needle := range []string{"| Tool", "Bundled Version", "PATH Version"} {
		if !strings.Contains(stdout, needle) {
			t.Errorf("stdout is missing the header %q", needle)
		}
	}
	rows := parseListRows(t, stdout)
	if len(rows) != len(registry) {
		t.Errorf("parsed %d rows, want %d", len(rows), len(registry))
	}
	for _, tool := range registry {
		row, ok := rows[tool.Name]
		if !ok {
			t.Errorf("no row for %q", tool.Name)
			continue
		}
		if row != [2]string{"-", "-"} {
			t.Errorf("%s = %q, want [- -]", tool.Name, row)
		}
	}
}

func TestListBundledVersionPrefersNewestHostBundle(t *testing.T) {
	emptyToolsDir(t)
	platform, err := PlatformDir()
	if err != nil {
		t.Fatalf("PlatformDir: %v", err)
	}
	executable := "shfmt"
	if isWindows() {
		executable += ".exe"
	}
	toolsDir := os.Getenv("FORMATTINGTON_FORMATTERS_DIR")
	makeBundledLayout(t, toolsDir, "shfmt", "3.13.1", platform, executable)
	makeBundledLayout(t, toolsDir, "shfmt", "3.14.1", platform, executable)
	// A bundle for another platform must not count: the cell is "-".
	makeBundledLayout(t, toolsDir, "black", "26.5.1", "not-a-platform", executable)
	// A newer bundle without the host executable must not count either.
	makeBundledLayout(t, toolsDir, "goimports", "9.9.9", platform, "notgoimports")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["shfmt"][0]; got != "3.14.1" {
		t.Errorf("shfmt bundled version = %q, want %q", got, "3.14.1")
	}
	if got := rows["black"][0]; got != "-" {
		t.Errorf("black bundled version = %q, want %q", got, "-")
	}
	if got := rows["goimports"][0]; got != "-" {
		t.Errorf("goimports bundled version = %q, want %q", got, "-")
	}
}

func TestListPathVersionIsProbed(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	binDir := os.Getenv("PATH")
	writeVersionFake(t, binDir, "shfmt", "shfmt version v3.7.0")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["shfmt"][1]; got != "3.7.0" {
		t.Errorf("shfmt PATH version = %q, want %q", got, "3.7.0")
	}
}

func TestListBothSides(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	platform, err := PlatformDir()
	if err != nil {
		t.Fatalf("PlatformDir: %v", err)
	}
	toolsDir := os.Getenv("FORMATTINGTON_FORMATTERS_DIR")
	executable := "clang-format"
	if isWindows() {
		executable += ".exe"
	}
	makeBundledLayout(t, toolsDir, "clang-format", "23.1.3", platform, executable)
	writeVersionFake(t, os.Getenv("PATH"), "clang-format", "clang-format version 22.1.8")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["clang-format"]; got != [2]string{"23.1.3", "22.1.8"} {
		t.Errorf("clang-format = %q, want [23.1.3 22.1.8]", got)
	}
}

func TestListPrefersSwiftFormatOverSwift(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	binDir := os.Getenv("PATH")
	writeVersionFake(t, binDir, "swift", "6.3.0")
	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["swift-format"][1]; got != "6.3.0" {
		t.Errorf("swift-format PATH version via swift = %q, want %q", got, "6.3.0")
	}

	writeVersionFake(t, binDir, "swift-format", "swift-format version 602.0.0")
	code, stdout, _ = runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows = parseListRows(t, stdout)
	if got := rows["swift-format"][1]; got != "602.0.0" {
		t.Errorf("swift-format PATH version = %q, want %q", got, "602.0.0")
	}
}

func TestListProbesGoBinariesViaGoVersion(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	binDir := os.Getenv("PATH")
	// The fake "go" ignores its arguments; make it report a Go toolchain.
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte("#!/bin/sh\nprintf 'gofmt: go9.9.9\\n'\n"), 0o755); err != nil {
		t.Fatalf("write go fake: %v", err)
	}
	writeVersionFake(t, binDir, "gofmt", "")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["gofmt"][1]; got != "9.9.9" {
		t.Errorf("gofmt PATH version = %q, want %q", got, "9.9.9")
	}
}

func TestListGoBinariesWithoutGoReportDash(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	writeVersionFake(t, os.Getenv("PATH"), "gofmt", "")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["gofmt"][1]; got != "-" {
		t.Errorf("gofmt PATH version without go = %q, want %q", got, "-")
	}
}

func TestListFailingProbeReportsDash(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	writeVersionFake(t, os.Getenv("PATH"), "black", "black, not-a-version")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["black"][1]; got != "-" {
		t.Errorf("black PATH version = %q, want %q", got, "-")
	}
}

func TestListPowershellFallsBackToPowershellOnWindows(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	binDir := os.Getenv("PATH")
	writeVersionFake(t, binDir, "powershell", "PowerShell 7.4.1")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["powershell"][1]; got != "7.4.1" {
		t.Errorf("powershell PATH version = %q, want %q", got, "7.4.1")
	}
}

func TestListZigUsesVersionSubcommand(t *testing.T) {
	requirePOSIX(t)
	emptyToolsDir(t)
	writeVersionFake(t, os.Getenv("PATH"), "zig", "0.17.0")

	code, stdout, _ := runCLI(t, "list")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	rows := parseListRows(t, stdout)
	if got := rows["zig-fmt"][1]; got != "0.17.0" {
		t.Errorf("zig-fmt PATH version = %q, want %q", got, "0.17.0")
	}
}

func TestFirstVersion(t *testing.T) {
	cases := map[string]string{
		"black, 26.5.1":                     "26.5.1",
		"prettier v3.3.3":                   "3.3.3",
		"PowerShell 7.4.1":                  "7.4.1",
		"asm 2026-11-14 x86_64":             "",
		"Mix 1.14.0 (compiled with OTP 25)": "1.14.0",
		"Dart SDK version: 3.5.0 (stable)":  "3.5.0",
		"clang-format version 23.1.3":       "23.1.3",
		"go version go1.27.3 darwin/amd64":  "1.27.3",
		"no version here":                   "",
		"":                                  "",
	}
	for input, want := range cases {
		if got := firstVersion(input); got != want {
			t.Errorf("firstVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGoToolchainVersion(t *testing.T) {
	cases := map[string]string{
		"/usr/local/bin/gofmt: go1.27.3":        "1.27.3",
		"gofmt: go1.22.1 darwin/arm64":          "1.22.1",
		"stat gofmt: no such file or directory": "",
	}
	for input, want := range cases {
		match := goToolchainVersion.FindStringSubmatch(input)
		got := ""
		if match != nil {
			got = match[1]
		}
		if got != want {
			t.Errorf("goToolchainVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestListHelpMentionsList(t *testing.T) {
	fakeTools(t)

	code, stdout, _ := runCLI(t, "--help")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "list") {
		t.Error("help output does not mention the list command")
	}
}

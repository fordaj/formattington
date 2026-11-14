package formattington

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// "formattington list" reports, for every registered tool, the version of
// the bundle Formattington would resolve for the host platform and the
// version of the tool found on PATH. A cell is "-" when that side does not
// exist: no bundle carries a host-platform executable, the tool is not on
// PATH, or the PATH version cannot be determined.

// listProbeTimeout bounds each version probe so a hanging tool cannot
// stall the whole report.
const listProbeTimeout = 10 * time.Second

// listCandidate describes one PATH-resolved executable used to report a
// tool, mirroring how the integration resolves it.
type listCandidate struct {
	// name is the executable to look up on PATH.
	name string
	// probeArgs are appended to the resolved executable to print its
	// version. A nil slice defaults to ["--version"].
	probeArgs []string
	// goVersion resolves the version via "go version <path>" instead of
	// invoking the candidate (gofmt and goimports carry no version flag).
	goVersion bool
}

// listCandidates lists the PATH candidates to try, in order, for a tool
// whose resolution differs from the defaults. Tools absent from the table
// resolve a single candidate named after the registry tool and probe with
// --version.
var listCandidates = map[string][]listCandidate{
	"powershell": {
		{name: "pwsh"},
		{name: "powershell"},
	},
	"styler": {
		// Rscript's own --version reports the R release, not the styler
		// package; ask R for the package version instead. When styler is
		// not installed the probe exits non-zero and the cell is "-".
		{name: "Rscript", probeArgs: []string{"-e", `cat(as.character(packageVersion("styler")))`}},
	},
	"swift-format": {
		{name: "swift-format"},
		{name: "swift", probeArgs: []string{"format", "--version"}},
	},
	"dart-format": {
		// "dart --version" prints to stderr.
		{name: "dart"},
	},
	"dotnet-format": {
		{name: "dotnet", probeArgs: []string{"format", "--version"}},
	},
	"mix-format": {
		{name: "mix"},
	},
	"zig-fmt": {
		{name: "zig", probeArgs: []string{"version"}},
	},
	"gofmt": {
		{name: "gofmt", goVersion: true},
	},
	"goimports": {
		{name: "goimports", goVersion: true},
	},
}

// candidatesFor returns the PATH candidates for name.
func candidatesFor(name string) []listCandidate {
	if candidates, ok := listCandidates[name]; ok {
		return candidates
	}
	return []listCandidate{{name: name}}
}

// versionPattern matches the first bare semantic version in tool output.
var versionPattern = regexp.MustCompile(`[0-9]+\.[0-9]+(\.[0-9]+)?`)

// goToolchainVersion matches the toolchain version "go version <path>"
// reports for a Go binary (for example "gofmt: go1.27.3" -> "1.27.3").
var goToolchainVersion = regexp.MustCompile(`go([0-9]+\.[0-9]+(\.[0-9]+)?)`)

// firstVersion returns the first bare semantic version in output, or "".
func firstVersion(output string) string {
	return versionPattern.FindString(output)
}

// outputWithTimeout runs executable and captures its combined output. A
// non-nil error means the executable could not be run or exceeded the
// timeout.
func outputWithTimeout(executable string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

// probeCandidate asks a resolved executable for its version and parses it,
// returning "" when it cannot be determined.
func probeCandidate(candidate listCandidate, resolved string) string {
	if candidate.goVersion {
		goBin := findOnPath("go")
		if goBin == "" {
			return ""
		}
		output, err := outputWithTimeout(goBin, []string{"version", resolved})
		if err != nil {
			return ""
		}
		if match := goToolchainVersion.FindStringSubmatch(output); match != nil {
			return match[1]
		}
		return ""
	}
	args := candidate.probeArgs
	if args == nil {
		args = []string{"--version"}
	}
	output, err := outputWithTimeout(resolved, args)
	if err != nil {
		return ""
	}
	return firstVersion(output)
}

// bundledVersionFor returns the version of the bundle FindTool would
// resolve for the host platform (the newest bundled version that carries a
// host-platform executable), or "" when no such bundle exists.
func bundledVersionFor(name string) string {
	platform, err := PlatformDir()
	if err != nil {
		return ""
	}
	executable := name
	if isWindows() {
		executable += ".exe"
	}
	root := FormattersDir()
	for _, version := range bundledVersions(filepath.Join(root, name)) {
		candidate := filepath.Join(root, name, "v"+version, platform, executable)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return version
		}
	}
	return ""
}

// pathVersionFor resolves the tool on PATH the same way its integration
// does and probes the first candidate found for its version, returning ""
// when the tool is absent or its version cannot be determined.
func pathVersionFor(name string) string {
	for _, candidate := range candidatesFor(name) {
		resolved := findOnPath(candidate.name)
		if resolved == "" {
			continue
		}
		return probeCandidate(candidate, resolved)
	}
	return ""
}

// printList writes the tool table to out.
func printList(out io.Writer) {
	type row [3]string
	cell := func(value string) string {
		if value == "" {
			return "-"
		}
		return value
	}

	table := []row{{"Tool", "Bundled Version", "PATH Version"}}
	for _, tool := range registry {
		table = append(table, row{tool.Name, cell(bundledVersionFor(tool.Name)), cell(pathVersionFor(tool.Name))})
	}

	widths := [3]int{}
	for _, r := range table {
		for i, value := range r {
			if len(value) > widths[i] {
				widths[i] = len(value)
			}
		}
	}

	printRow := func(r row) {
		fmt.Fprintf(out, "| %-*s | %-*s | %-*s |\n", widths[0], r[0], widths[1], r[1], widths[2], r[2])
	}
	printRow(table[0])
	fmt.Fprintf(out, "| %s | %s | %s |\n",
		strings.Repeat("-", widths[0]), strings.Repeat("-", widths[1]), strings.Repeat("-", widths[2]))
	for _, r := range table[1:] {
		printRow(r)
	}
}

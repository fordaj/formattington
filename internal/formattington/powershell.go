package formattington

import (
	"fmt"
	"path/filepath"
	"strings"
)

// powershellExtensions are the file extensions handled by the PowerShell
// integration.
var powershellExtensions = []string{".ps1", ".psm1"}

// powerShellShells are the PowerShell hosts tried in order: PowerShell 7+
// ("pwsh") is required for PSScriptAnalyzer; on Windows the in-box Windows
// PowerShell ("powershell.exe") is tried as a fallback.
var powerShellShells = []string{"pwsh", "powershell"}

// findPowerShellShell resolves "pwsh" (preferred) or "powershell".
//
// The error message follows the FindTool convention so the CLI prints a
// single "formattington: Could not find ..." line and exits 127.
func findPowerShellShell(verbose bool) (string, error) {
	for _, candidate := range powerShellShells {
		executable, err := FindTool(candidate, "", verbose)
		if err == nil {
			return executable, nil
		}
	}
	return "", fmt.Errorf(
		"Could not find 'pwsh' or 'powershell'. " +
			"Expected PowerShell 7+ (pwsh) on PATH and the PSScriptAnalyzer " +
			"module ('Install-Module PSScriptAnalyzer' inside pwsh).",
	)
}

// isPowerShellPathArg reports whether a forwarded token is a PowerShell
// file to analyze. This is intentionally suffix-only (unlike
// hasUserFileArgs, which also checks existence for discovery suppression).
func isPowerShellPathArg(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}
	suffix := strings.ToLower(filepath.Ext(arg))
	return suffix == ".ps1" || suffix == ".psm1"
}

// singleQuote quotes a path for a PowerShell single-quoted string.
func singleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// runPowerShell lints PowerShell files with PSScriptAnalyzer (no in-place
// formatting).
//
// PSScriptAnalyzer is a PowerShell module run through "pwsh -NoProfile
// -Command ...". It reports findings but never rewrites files, so every
// mode behaves the same: findings exit 1, no findings exit 0. Check and
// DryRun are accepted for CLI uniformity; they do not change the lint
// invocation.
//
// Files (".ps1"/".psm1" paths) are passed via "-Path"; any other forwarded
// token becomes an Invoke-ScriptAnalyzer parameter, so e.g.
// "formattington ps1 -Severity Error" forwards "-Severity" and "Error".
// With no files, the current directory is analyzed recursively.
func runPowerShell(opts Options) (int, error) {
	executable, err := findPowerShellShell(opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without invoking the analyzer.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	var paths, params []string
	for _, arg := range forwarded {
		if isPowerShellPathArg(arg) {
			paths = append(paths, arg)
			continue
		}
		params = append(params, arg)
	}

	pathList := "'.'"
	recurseSuffix := ""
	if len(paths) > 0 {
		quoted := make([]string, 0, len(paths))
		for _, path := range paths {
			quoted = append(quoted, singleQuote(path))
		}
		pathList = strings.Join(quoted, ", ")
	} else {
		recurseSuffix = " -Recurse"
	}
	paramsSuffix := ""
	if len(params) > 0 {
		paramsSuffix = " " + strings.Join(params, " ")
	}

	script := "Import-Module PSScriptAnalyzer -ErrorAction Stop; " +
		"$issues = @(Invoke-ScriptAnalyzer -Path " + pathList + recurseSuffix + paramsSuffix + "); " +
		"$issues; if ($issues.Count -gt 0) { exit 1 }"

	return runTool(executable, []string{"-NoProfile", "-Command", script}, false)
}

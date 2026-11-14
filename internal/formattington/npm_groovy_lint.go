package formattington

// npmGroovyLintExtensions are the file extensions handled by
// npm-groovy-lint.
var npmGroovyLintExtensions = []string{".groovy", ".gradle"}

// runNpmGroovyLint formats Groovy source files using npm-groovy-lint
// (github.com/nvuillam/npm-groovy-lint; install with
// "npm install -g npm-groovy-lint"), the only viable Groovy formatter:
// Spotless is a Gradle/Maven build plugin with no standalone CLI, and
// Apache Groovy ships no formatter. Layout-only — it fixes
// indentation/brace spacing/blank lines and never re-breaks long lines.
//
// Requirements: Node >= 18 and Java 11+ on PATH (a long-lived Java
// microservice backs the analysis; "--noserver" disables it).
//
// Check mode is linter-flavored and is NOT a byte-comparison: the tool
// reports layout rule violations at or above "--failon" and exits 1 at
// the threshold. It intentionally does not rewrite files in check mode.
// For any layout findings to appear the project's .groovylintrc must
// enable the "format" preset (for example: extends: ["recommended",
// "format"]), since current versions do not report layout by default.
// Jenkinsfile (extensionless) is out of scope for discovery; pass it
// explicitly or via a config command.
//
// Args are forwarded verbatim; ".groovylintrc.js/.json/.yml" discovery
// is tool-native, or pass "-c" yourself.
func runNpmGroovyLint(opts Options) (int, error) {
	executable, err := FindTool("npm-groovy-lint", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	switch {
	case opts.Check:
		// Linter-flavored check: no writes, exit 1 at the error threshold.
		return runTool(executable, withArgs([]string{"--failon", "error"}, forwarded), false)
	case opts.DryRun:
		// No diff mode exists; same non-modifying invocation.
		return runTool(executable, withArgs([]string{"--failon", "error"}, forwarded), false)
	default:
		// Format in place (layout only).
		return runTool(executable, withArgs([]string{"--format"}, forwarded), false)
	}
}

package formattington

// elmFormatExtensions are the file extensions handled by elm-format.
var elmFormatExtensions = []string{".elm"}

// runElmFormat formats Elm files using elm-format.
//
// elm-format is deliberately zero-configuration (gofmt philosophy) —
// there is no config file, and none must be invented. "--yes" suppresses
// the interactive overwrite prompt (all overwrites accepted). "--validate"
// performs a non-writing check and exits 0 when files are formatted, 1
// otherwise (error paths also exit 1 — the report itself disambiguates).
// There is no diff mode, so --dry-run maps to the same --validate
// invocation. Arguments are forwarded verbatim; "--elm-version" is an
// example.
func runElmFormat(opts Options) (int, error) {
	executable, err := FindTool("elm-format", "", opts.Verbose)
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
		// Fail if any file needs formatting; don't modify files.
		return runTool(executable, withArgs([]string{"--validate"}, forwarded), false)
	case opts.DryRun:
		// Validate without modifying files (no diff mode exists).
		return runTool(executable, withArgs([]string{"--validate"}, forwarded), false)
	default:
		// Format in place, suppressing the interactive prompt.
		return runTool(executable, withArgs([]string{"--yes"}, forwarded), false)
	}
}

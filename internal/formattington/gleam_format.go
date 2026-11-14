package formattington

// gleamFormatExtensions are the file extensions handled by gleam format.
var gleamFormatExtensions = []string{".gleam"}

// runGleamFormat formats Gleam files using "gleam format" (built into
// the Gleam compiler; install via the install script, Homebrew, Nix, or
// Cargo).
//
// Check: "gleam format --check <files...>" prints the files that would
// change and exits non-zero (Error::Format, i.e. 1) when any would —
// reported as ExitFormatRequired. There is no diff mode; --dry-run maps
// to the same --check invocation. No configuration by design (fixed
// style), and none is invented. Arguments are forwarded verbatim.
func runGleamFormat(opts Options) (int, error) {
	executable, err := FindTool("gleam", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, withArgs([]string{"format"}, forwarded), false)
	}

	switch {
	case opts.Check, opts.DryRun:
		// Fail if any file needs formatting; don't modify files.
		args := append([]string{"format", "--check"}, forwarded...)
		code, runErr := runTool(executable, args, false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code != ExitOK {
			return ExitFormatRequired, nil
		}
		return ExitOK, nil
	default:
		// Format in place.
		args := append([]string{"format"}, forwarded...)
		return runTool(executable, args, false)
	}
}

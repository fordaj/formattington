package formattington

// fprettifyExtensions are the file extensions handled by fprettify.
//
// Free-form Fortran only. Fixed-form ".f"/".for" sources are not
// registered until fprettify's handling of them is verified.
var fprettifyExtensions = []string{".f90", ".f95", ".f03", ".f08", ".f18"}

// runFprettify formats Fortran files using fprettify (PATH-only; fprettify
// is GPL-3.0+ and pure Python — embedding it into the release binaries was
// rejected for licensing/packaging reasons, so it must be installed
// separately, e.g. "pipx install fprettify").
//
// fprettify has no native check mode and its "--diff" output does not
// affect the exit code. Check mode is therefore synthesized: fprettify is
// run with "--diff" and the output is captured; non-empty output means
// files would be reformatted and is reported as ExitFormatRequired (with
// the diff itself printed). Empty output means clean. A non-zero exit
// from the tool passes through untouched. Dry-run prints the diff without
// capturing anything. Arguments are forwarded verbatim ("-c/--config-file"
// selects a configuration file; ".fprettify.rc" discovery is tool-native).
func runFprettify(opts Options) (int, error) {
	executable, err := FindTool("fprettify", "", opts.Verbose)
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
		// Synthesized check: non-empty --diff output means formatting
		// changes are required (runTool reports this as ExitFormatRequired).
		return runTool(executable, withArgs([]string{"--diff"}, forwarded), true)
	case opts.DryRun:
		// Print the diff without modifying files.
		return runTool(executable, withArgs([]string{"--diff"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

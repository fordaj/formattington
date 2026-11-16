package formattington

// shfmtExtensions are the file extensions handled by shfmt.
var shfmtExtensions = []string{".sh", ".bash"}

// runShfmt formats shell scripts with shfmt.
//
// Formattington owns the mode; shfmt owns all other flags.
func runShfmt(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("shfmt", opts, nil)
	if err != nil || done {
		return code, err
	}

	switch {
	case opts.Check:
		// "-l" lists the files that need formatting; the exit code is
		// derived from the captured output.
		return runTool(executable, withArgs([]string{"-l"}, forwarded), true)
	case opts.DryRun:
		return runTool(executable, withArgs([]string{"-d"}, forwarded), false)
	default:
		return runTool(executable, withArgs([]string{"-w"}, forwarded), false)
	}
}

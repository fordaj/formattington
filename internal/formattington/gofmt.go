package formattington

// gofmtExtensions are the file extensions handled by gofmt.
var gofmtExtensions = []string{".go"}

// runGofmt formats Go files using gofmt.
//
// "gofmt -l" lists files whose formatting differs but always exits 0, so
// the output is captured and reported as ExitFormatRequired when it is
// non-empty (like shfmt "-l").
func runGofmt(opts Options) (int, error) {
	executable, err := FindTool("gofmt", "", opts.Verbose)
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
		return runTool(executable, withArgs([]string{"-l"}, forwarded), true)
	case opts.DryRun:
		// Show diffs without modifying files.
		return runTool(executable, withArgs([]string{"-d"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"-w"}, forwarded), false)
	}
}

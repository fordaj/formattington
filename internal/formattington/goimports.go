package formattington

// goimportsExtensions are the file extensions handled by goimports.
var goimportsExtensions = []string{".go"}

// runGoimports formats Go files using goimports (gofmt + import fixing).
//
// goimports formats like gofmt and additionally fixes import grouping:
// missing imports are added, unused imports are removed, and imports are
// grouped into sections. Individual import options (for example "-local")
// are forwarded verbatim and are not overridden.
func runGoimports(opts Options) (int, error) {
	executable, err := FindTool("goimports", "", opts.Verbose)
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
		// Format in place (including import fixes).
		return runTool(executable, withArgs([]string{"-w"}, forwarded), false)
	}
}

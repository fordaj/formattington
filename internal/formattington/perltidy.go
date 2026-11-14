package formattington

// perltidyExtensions are the file extensions handled by perltidy.
var perltidyExtensions = []string{".pl", ".pm", ".t"}

// runPerltidy formats Perl files using perltidy (PATH-only; perltidy is
// GPL-2.0+ and is never embedded into the release binaries).
//
// Check mode uses the native "perltidy --assert-tidy" flag, which exits 0
// when the input is tidy and non-zero otherwise; any non-zero code is
// reported as ExitFormatRequired and stderr is preserved. (Very old
// perltidy releases without --assert-tidy fail with a usage error, which
// is also reported as ExitFormatRequired; the manual recipe is
// "perltidy -st file | diff - file".) Dry-run prints a diff with "-d" and
// does not modify files. The default rewrites files in place with "-b"
// (backup), using "-bext=/" to suppress the ".bak" files perltidy would
// otherwise leave next to every formatted file. Arguments are forwarded
// verbatim ("--profile=path", "--noprofile", "-i=4", ...); configuration
// (".perltidyrc") is discovered by the tool itself.
func runPerltidy(opts Options) (int, error) {
	executable, err := FindTool("perltidy", "", opts.Verbose)
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
		// Native check mode: exit 0 when tidy, non-zero otherwise.
		code, runErr := runTool(executable, withArgs([]string{"--assert-tidy"}, forwarded), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code != ExitOK {
			return ExitFormatRequired, nil
		}
		return ExitOK, nil
	case opts.DryRun:
		// Print a diff without modifying files.
		return runTool(executable, withArgs([]string{"-d"}, forwarded), false)
	default:
		// Format in place; suppress the .bak files perltidy writes by default.
		return runTool(executable, withArgs([]string{"-b", "-bext=/"}, forwarded), false)
	}
}

package formattington

// cljfmtExtensions are the file extensions handled by cljfmt.
var cljfmtExtensions = []string{".clj", ".cljs", ".cljc", ".edn"}

// runCljfmt formats Clojure files using cljfmt
// (github.com/weavejester/cljfmt, distributed under EPL-1.0).
//
// cljfmt's CLI is subcommand-shaped ("cljfmt fix" / "cljfmt check"), not
// flag-shaped. "cljfmt check" prints a unified diff and exits 1 when
// files need formatting (0 when clean, 2 on parse/format error); that is
// both the check mapping and the dry-run mapping (there is no separate
// diff mode). "cljfmt fix" rewrites files in place. Exit 2 passes
// through verbatim — it is never conflated with exit 1. Args are
// forwarded verbatim ("--config FILE" is honored); configuration is
// discovered by the tool itself (".cljfmt.edn"/"cljfmt.edn" in cwd and
// parents).
func runCljfmt(opts Options) (int, error) {
	executable, err := FindTool("cljfmt", "", opts.Verbose)
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
		return runTool(executable, withArgs([]string{"check"}, forwarded), false)
	case opts.DryRun:
		// cljfmt check IS the diff mode.
		return runTool(executable, withArgs([]string{"check"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"fix"}, forwarded), false)
	}
}

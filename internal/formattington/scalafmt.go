package formattington

// scalafmtExtensions are the file extensions handled by scalafmt.
var scalafmtExtensions = []string{".scala", ".sbt", ".sc"}

// runScalafmt formats Scala files using scalafmt (PATH-only; install with
// "cs install scalafmt" or "brew install scalafmt").
//
// scalafmt requires a "version = <x.y.z>" line in .scalafmt.conf (since
// 3.1.0). The upstream native release binaries have no dynamic runner, so
// a config whose pinned version differs from the compiled-in one fails
// hard ("invalid Scalafmt version ... this error happens only when
// running a native Scalafmt binary", exit bit 16). Formattington therefore
// does NOT bundle the natives: users get scalafmt's JVM/coursier launcher
// from PATH, which honors the pinned version by downloading it
// dynamically — the only fidelity route.
//
// "--test" prints the diff and exits 1 when any file would change (0 when
// clean, no writes) — that is both the check mapping and the dry-run
// mapping (scalafmt has no separate diff mode; "--test" is it). Exit
// codes are a bitfield (Ok=0, TestError=1, ParseError=2, CmdLineArg=4,
// Unexpected=8, UnsupportedVersion=16, OR-ed); bit 16 surfaces whenever a
// native scalafmt binary runs against a differently-pinned config. All
// untouched bits pass through verbatim. Args are forwarded verbatim
// ("--config" is honored); a config is never synthesized.
func runScalafmt(opts Options) (int, error) {
	executable, err := FindTool("scalafmt", "", opts.Verbose)
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
		return runTool(executable, withArgs([]string{"--test"}, forwarded), false)
	case opts.DryRun:
		// scalafmt has no separate diff mode; --test is it.
		return runTool(executable, withArgs([]string{"--test"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

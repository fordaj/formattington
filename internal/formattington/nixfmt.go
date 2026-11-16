package formattington

// nixfmtExtensions are the file extensions handled by nixfmt.
var nixfmtExtensions = []string{".nix"}

// runNixfmt formats Nix files using nixfmt (RFC 166 "rfc style").
//
// "nixfmt --check" exits 1 when any file is not formatted (exit 1 is also
// reported on parse errors, indistinguishable from formatting findings; the
// stderr output disambiguates). There is no diff mode, so --dry-run maps to
// the same "--check" invocation. Arguments are forwarded verbatim; nixfmt
// takes no configuration file ("--width"/"--indent" exist as flags only).
func runNixfmt(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("nixfmt", opts, nil)
	if err != nil || done {
		return code, err
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		return runTool(executable, withArgs([]string{"--check"}, forwarded), false)
	case opts.DryRun:
		// nixfmt has no diff mode; validate without modifying files.
		return runTool(executable, withArgs([]string{"--check"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

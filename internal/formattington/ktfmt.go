package formattington

// ktfmtExtensions are the file extensions handled by ktfmt.
var ktfmtExtensions = []string{".kt", ".kts"}

// runKtfmt formats Kotlin files using ktfmt (github.com/Kotlin/ktfmt).
//
// Check mode is "ktfmt --dry-run --set-exit-if-changed <files...>": exit 1
// when any file would change, otherwise 0 (no writes). A bare "--dry-run"
// prints the files that would change but always exits 0 — it is not a
// usable check, so --check fails lint via --set-exit-if-changed. No
// style flag is ever injected: --google-style / --kotlinlang-style /
// --meta-style and friends are forwarded verbatim (the upstream default
// is --kotlinlang-style). ktfmt has no configuration file by design;
// ".editorconfig" is honored only with --enable-editorconfig.
func runKtfmt(opts Options) (int, error) {
	executable, err := FindTool("ktfmt", "", opts.Verbose)
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
		return runTool(executable, withArgs([]string{"--dry-run", "--set-exit-if-changed"}, forwarded), false)
	case opts.DryRun:
		// List files that would change; upstream always exits 0 here.
		return runTool(executable, withArgs([]string{"--dry-run"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

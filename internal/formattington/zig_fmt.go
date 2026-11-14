package formattington

// zigFmtExtensions are the file extensions handled by zig fmt.
//
// ".zon" (ZON) is deliberately NOT registered: it is an opt-in flag
// upstream; pass "--zon" verbatim to enable it.
var zigFmtExtensions = []string{".zig"}

// runZigFmt formats Zig files using "zig fmt" (ships with every Zig
// release).
//
// Zig is pre-1.0, so formatting can drift between toolchain versions —
// Formattington always uses the project's own zig (whatever is on PATH),
// which matches the project's toolchain by construction.
//
// Check: "zig fmt --check <files...>" exits 1 when any file would change
// (missing files also exit 1 with a stderr diagnostic; the diagnostic
// disambiguates). There is no diff mode, so --dry-run maps to the same
// --check invocation. Zero configuration by design — no config file
// exists and none is invented. Arguments are forwarded verbatim.
func runZigFmt(opts Options) (int, error) {
	executable, err := FindTool("zig", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, withArgs([]string{"fmt"}, forwarded), false)
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		return runTool(executable, withArgs([]string{"fmt", "--check"}, forwarded), false)
	case opts.DryRun:
		// No diff mode exists; validate without modifying files.
		return runTool(executable, withArgs([]string{"fmt", "--check"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"fmt"}, forwarded), false)
	}
}

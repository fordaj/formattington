package formattington

// erlfmtExtensions are the file extensions handled by erlfmt.
var erlfmtExtensions = []string{".erl", ".hrl"}

// runErlfmt formats Erlang source files using erlfmt
// (github.com/WhatsApp/erlfmt), the mature Erlang formatter. There is no
// bundle: erlfmt ships as source (a rebar3 plugin or an escript built from
// it) and requires a working Erlang/OTP 21+ install, so it is resolved from
// PATH only (either a standalone 'erlfmt' escript or erlfmt made
// available via the project's rebar3 build).
//
// "erlfmt -c" exits 1 when any file needs formatting (0 otherwise); there
// is no diff mode, so --dry-run maps to the same check invocation. Args
// are forwarded verbatim. erlfmt takes no configuration file by design.
func runErlfmt(opts Options) (int, error) {
	executable, err := FindTool("erlfmt", "", opts.Verbose)
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
		return runTool(executable, withArgs([]string{"-c"}, forwarded), false)
	case opts.DryRun:
		// erlfmt has no diff mode; validate without modifying files.
		return runTool(executable, withArgs([]string{"-c"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"-w"}, forwarded), false)
	}
}

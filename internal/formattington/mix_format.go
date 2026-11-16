package formattington

// mixFormatExtensions are the file extensions handled by mix format.
var mixFormatExtensions = []string{".ex", ".exs"}

// runMixFormat formats Elixir files using "mix format" (ships with the
// Elixir installation; needs Erlang/OTP on PATH).
//
// The tool name is "mix-format" — the bare binary "mix" would collide
// with the build tool's own command shape. The "elixir" alias resolves
// to it.
//
// Default invocation is "mix format --no-compile <files...>": formatter
// plugins are skipped so formatting never compiles the project. Drop
// "--no-compile" yourself (or set formatter plugins in .formatter.exs)
// when plugins are required. Check is the same invocation with
// "--check-formatted", which exits non-zero when any file would change —
// reported as ExitFormatRequired. There is no diff mode; --dry-run maps
// to the same --check-formatted invocation.
//
// WARNING: the project's .formatter.exs is EXECUTED CODE (returns a
// keyword list: inputs, plugins, import_deps, ...), discovered from the
// CURRENT DIRECTORY (not walked up). Do not run this against untrusted
// projects. Formatter output can also change between Elixir versions.
// Args are forwarded verbatim.
func runMixFormat(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("mix", opts, []string{"format"})
	if err != nil || done {
		return code, err
	}

	switch {
	case opts.Check, opts.DryRun:
		// Fail if any file needs formatting; don't modify files.
		args := append([]string{"format", "--check-formatted", "--no-compile"}, forwarded...)
		code, runErr := runTool(executable, args, false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code != ExitOK {
			return ExitFormatRequired, nil
		}
		return ExitOK, nil
	default:
		// Format in place (plugins skipped).
		args := append([]string{"format", "--no-compile"}, forwarded...)
		return runTool(executable, args, false)
	}
}

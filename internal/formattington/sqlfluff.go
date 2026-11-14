package formattington

// sqlfluffExtensions are the file extensions handled by SQLFluff.
var sqlfluffExtensions = []string{".sql"}

// runSqlfluff lints and formats SQL files using SQLFluff (PATH-only;
// install with "pipx install sqlfluff" / "pip install sqlfluff").
//
// SQLFluff is invoked with "--disable-progress-bar" so captured output is
// deterministic. The SQL dialect is never hardcoded: when the project has
// a ".sqlfluff" config (or [tool.sqlfluff] in pyproject.toml) it resolves
// the dialect there; otherwise SQLFluff falls back to "ansi" — set a
// dialect in the project config (e.g. "dialect = postgres") or forward
// "--dialect" yourself (e.g. via a config command).
//
// "sqlfluff lint" exits 1 on violations, 2 on error; exit 1 is reported as
// ExitFormatRequired and exit 2 passes through untouched. "sqlfluff fix"
// rewrites files in place. "--dry-run --diff" prints a diff without
// writing (git-style patch via SQLFluff). Arguments are forwarded
// verbatim; "--config", "--dialect" and friends all work natively.
func runSqlfluff(opts Options) (int, error) {
	executable, err := FindTool("sqlfluff", "", opts.Verbose)
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
		// Check mode: lint without modifying files. Exit 1 means findings.
		code, runErr := runTool(executable, withArgs([]string{"lint", "--disable-progress-bar"}, forwarded), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code == ExitFormatRequired {
			return ExitFormatRequired, nil
		}
		return code, nil
	case opts.DryRun:
		// Show a diff without modifying files.
		return runTool(executable, withArgs([]string{"fix", "--disable-progress-bar", "--dry-run", "--diff"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"fix", "--disable-progress-bar"}, forwarded), false)
	}
}

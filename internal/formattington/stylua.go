package formattington

// styluaExtensions are the file extensions handled by StyLua.
var styluaExtensions = []string{".lua"}

// runStylua formats Lua files using StyLua.
//
// "stylua --check" compares a diff between all input files and exits 1
// when any file is not formatted; input files are not overwritten. The
// "--output-format=unified" flag is only honored together with "--check",
// so the dry-run mapping is "--check --output-format=unified", which
// prints a unified diff instead of the standard report. Arguments are
// forwarded verbatim; configuration ("stylua.toml"/".stylua.toml", or
// ".editorconfig" unless "--no-editorconfig" is passed) is discovered by
// the tool itself.
func runStylua(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("stylua", opts, nil)
	if err != nil || done {
		return code, err
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		return runTool(executable, withArgs([]string{"--check"}, forwarded), false)
	case opts.DryRun:
		// Print unified diffs without modifying files.
		return runTool(executable, withArgs([]string{"--check", "--output-format=unified"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

package formattington

// blackExtensions are the file extensions handled by Black.
var blackExtensions = []string{".py"}

// runBlack formats Python files using Black.
//
// Black respects its native configuration (for example "[tool.black]" in
// pyproject.toml); no "--line-length" flag is added unless the caller
// forwards one via Args. Black is resolved via FindTool (bundled lookup
// then PATH); it is installed via pip and is not bundled.
func runBlack(opts Options) (int, error) {
	executable, err := FindTool("black", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	arguments := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding formatting flags.
	if IsInformational(arguments) {
		return runTool(executable, arguments, false)
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		arguments = withArgs([]string{"--check"}, arguments)
	case opts.DryRun:
		// Show diff without modifying files.
		arguments = withArgs([]string{"--diff"}, arguments)
	}
	// Default: Black writes in place without a flag.

	return runTool(executable, arguments, false)
}

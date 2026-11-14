package formattington

// clangFormatExtensions are the file extensions handled by clang-format.
var clangFormatExtensions = []string{
	".c",
	".cc",
	".cpp",
	".cxx",
	".c++",
	".h",
	".hh",
	".hpp",
	".hxx",
}

// runClangFormat formats C/C++ files using clang-format.
func runClangFormat(opts Options) (int, error) {
	executable, err := FindTool("clang-format", "", opts.Verbose)
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
		arguments = withArgs([]string{"--dry-run", "--Werror"}, arguments)
	case opts.DryRun:
		// Validate without modifying files.
		arguments = withArgs([]string{"--dry-run"}, arguments)
	default:
		// Formattington formats in place by default.
		if !contains(arguments, "-i") && !contains(arguments, "--in-place") {
			arguments = withArgs([]string{"-i"}, arguments)
		}
	}

	return runTool(executable, arguments, false)
}

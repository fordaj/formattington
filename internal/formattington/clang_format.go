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
	executable, arguments, done, code, err := beginIntegration("clang-format", opts, nil)
	if err != nil || done {
		return code, err
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

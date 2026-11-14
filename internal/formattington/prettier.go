package formattington

// prettierExtensions are the file extensions handled by Prettier.
//
// ".md" and ".markdown" are included because Prettier also formats
// Markdown; Prettier is the sole owner of Markdown formatting.
var prettierExtensions = []string{
	".js",
	".ts",
	".tsx",
	".jsx",
	".json",
	".html",
	".css",
	".scss",
	".yaml",
	".yml",
	".md",
	".markdown",
}

// prettierWriteFlags are the forwarded flags that already select a mode, so
// the default "--write" must not be added on top of them.
var prettierWriteFlags = []string{
	"--write",
	"-w",
	"--check",
	"-c",
	"--list-different",
	"-l",
}

// runPrettier formats JS/TS/CSS/JSON/HTML/YAML/Markdown files using
// Prettier.
//
// Prettier has no unified-diff flag, so the dry-run equivalent is
// "--list-different", which lists files that differ from Prettier
// formatting without modifying them.
func runPrettier(opts Options) (int, error) {
	executable, err := FindTool("prettier", "", opts.Verbose)
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
		// List files that differ without modifying files.
		arguments = withArgs([]string{"--list-different"}, arguments)
	default:
		// Formattington formats in place by default, unless the caller
		// already selected a mode.
		selected := false
		for _, flag := range prettierWriteFlags {
			if contains(arguments, flag) {
				selected = true
				break
			}
		}
		if !selected {
			arguments = withArgs([]string{"--write"}, arguments)
		}
	}

	return runTool(executable, arguments, false)
}

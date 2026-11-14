package formattington

// gdformatExtensions are the file extensions handled by gdformat.
var gdformatExtensions = []string{".gd"}

// runGdformat formats Godot 4 GDScript files using gdformat
// (github.com/Scony/godot-gdscript-toolkit, the gdtoolkit Python package,
// install with "pipx install gdtoolkit==4.*" for the Godot 4 line).
//
// There is no bundle: gdtoolkit is a pure-Python package with no release
// binaries, so it is PATH-only on all six targets.
//
// gdformat's check mode is "-c"/"--check": it prints "would reformat
// <file>" and exits 1 when changes are needed, 0 otherwise. "-d"/"--diff"
// prints a unified diff and implies --check. Arguments are forwarded
// verbatim; configuration ("gdformatrc" / ".gdformatrc", discovered by
// walking up from the working directory) is tool-native. The safety-checks
// prologue/epilogue validation is never disabled by Formattington ("--fast"
// is deliberately not set).
func runGdformat(opts Options) (int, error) {
	executable, err := FindTool("gdformat", "", opts.Verbose)
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
		// Print a diff; implies --check.
		return runTool(executable, withArgs([]string{"-d"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

package formattington

// fantomasExitCheck is the exit code Fantomas reports when files need
// formatting ("--check" mode; verified upstream: 0 clean, 1 error,
// 99 formatting needed). It is remapped to ExitFormatRequired; every
// other code passes through untouched.
const fantomasExitCheck = 99

// fantomasExtensions are the file extensions handled by Fantomas.
var fantomasExtensions = []string{".fs", ".fsi", ".fsx"}

// runFantomas formats F# files using Fantomas (github.com/fsprojects/fantomas),
// the F# toolchain formatter. No bundle exists: Fantomas ships only as a
// .NET global tool (NuGet nupkgs), and embedding the .NET runtime is out
// of scope, so it is PATH-only ("dotnet tool install -g fantomas").
//
// Fantomas formats in place by default; "--check" never writes and exits
// 99 when files need formatting (1 on error). There is no diff mode, so
// --dry-run maps to the same --check invocation. Arguments are forwarded
// verbatim; ".editorconfig" is the configuration source (many options
// are picked up automatically), with ".fantomasignore" for exclusions —
// both are discovered by the tool itself.
func runFantomas(opts Options) (int, error) {
	executable, err := FindTool("fantomas", "", opts.Verbose)
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
		code, runErr := runTool(executable, withArgs([]string{"--check"}, forwarded), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code == fantomasExitCheck {
			return ExitFormatRequired, nil
		}
		return code, nil
	case opts.DryRun:
		// No diff mode exists; validate without modifying files.
		code, runErr := runTool(executable, withArgs([]string{"--check"}, forwarded), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code == fantomasExitCheck {
			return ExitFormatRequired, nil
		}
		return code, nil
	default:
		// Format in place.
		return runTool(executable, forwarded, false)
	}
}

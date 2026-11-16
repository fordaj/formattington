package formattington

import "strings"

// dotnetFormatExtensions are the file extensions handled by dotnet format.
var dotnetFormatExtensions = []string{".cs", ".vb"}

// isDotnetFormatOptIn reports whether the caller's first forwarded
// argument selects an explicit dotnet-format mode: a subcommand
// ("whitespace", "style", "analyzers") or a project/solution path
// (".sln", ".csproj", ".vbproj", ".fsproj").
func isDotnetFormatOptIn(arg string) bool {
	switch arg {
	case "whitespace", "style", "analyzers":
		return true
	}
	lower := strings.ToLower(arg)
	for _, suffix := range []string{".sln", ".csproj", ".vbproj", ".fsproj"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// dotnetFormatValuedFlags are dotnet-format flags that consume the
// following argument as their value; they must not be mistaken for a
// file path. ("--diagnostics" takes a space-separated list and cannot
// be distinguished generically — users wanting it should use a config
// command.)
var dotnetFormatValuedFlags = map[string]bool{
	"--binarylog": true,
	"--exclude":   true,
	"--framework": true,
	"--include":   true,
	"--report":    true,
	"--severity":  true,
	"-v":          true,
	"--verbosity": true,
}

// dotnetVerifyExitCode is the exit code dotnet format --verify-no-changes
// reports when files would be formatted. It is remapped to
// ExitFormatRequired.
const dotnetVerifyExitCode = 2

// runDotnetFormat formats C# and Visual Basic .NET files using
// "dotnet format" (ships in-box with the .NET SDK ≥ 6).
//
// Invocation uses the "whitespace" subcommand with "--folder ." and
// "--include <files...>": whitespace rules only, no MSBuild project or
// solution required, and the include list is the discovered file list
// (paths relative to the --folder, i.e. the cwd). This is the shape
// suited to Formattington's file-list model. The "style" and "analyzers"
// subcommands can restore and build project code (implicit restore,
// executing analyzers against trusted-code boundaries); they are
// opted into by making the first forwarded argument a subcommand or a
// project/solution path, which switches the invocation to verbatim mode:
// e.g. "formattington dotnet-format style --severity info src/app.csproj".
//
// "--verify-no-changes" (check mode) exits 2 when any file would be
// formatted; it is remapped to ExitFormatRequired. There is no diff mode;
// --dry-run maps to the same --verify-no-changes invocation. Arguments
// are forwarded verbatim; ".editorconfig" is the only configuration
// source (discovered by the tool); "--report <dir>" emits JSON reports.
//
// When discovery finds nothing, the tool is still invoked once with
// "format --version" (a harmless no-op) to preserve the suite-wide
// contract that the selected tool runs.
func runDotnetFormat(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("dotnet", opts, []string{"format"})
	if err != nil || done {
		return code, err
	}

	files, flags := splitArgsAndFiles(forwarded, dotnetFormatValuedFlags)

	// Verbatim opt-in: if the caller's first argument is a dotnet-format
	// subcommand or a project/solution path, their command line is used
	// as-is (with --verify-no-changes appended in check mode) instead of
	// the whitespace/folder default. This is how style/analyzers modes —
	// which restore and compile project code — are opted into.
	if len(forwarded) > 0 && isDotnetFormatOptIn(forwarded[0]) {
		args := append([]string{"format"}, forwarded...)
		if opts.Check || opts.DryRun {
			args = append(args, "--verify-no-changes")
		}
		code, runErr := runTool(executable, args, false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if (opts.Check || opts.DryRun) && code == dotnetVerifyExitCode {
			return ExitFormatRequired, nil
		}
		return code, nil
	}

	args := []string{"format", "whitespace"}
	if len(files) == 0 && len(flags) == 0 {
		// Nothing discovered and no args: invoke once with a harmless
		// no-op to preserve the suite-wide invocation contract.
		return runTool(executable, []string{"format", "--version"}, false)
	}
	if !hasFlag(forwarded, "--folder") {
		args = append(args, "--folder", ".")
	}
	if len(files) > 0 && !hasFlag(forwarded, "--include") {
		args = append(args, "--include")
		args = append(args, files...)
	}
	args = append(args, flags...)

	switch {
	case opts.Check || opts.DryRun:
		args = append(args, "--verify-no-changes")
	}

	code, runErr := runTool(executable, args, false)
	if runErr != nil {
		return ExitNotFound, runErr
	}
	if (opts.Check || opts.DryRun) && code == dotnetVerifyExitCode {
		return ExitFormatRequired, nil
	}
	return code, nil
}

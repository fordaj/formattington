package formattington

// dartFormatExtensions are the file extensions handled by dart format.
var dartFormatExtensions = []string{".dart"}

// runDartFormat formats Dart files using "dart format" (dart-lang/sdk,
// built into the Dart SDK).
//
// The binary is "dart", invoked as "dart format"; the Formattington tool
// name is "dart-format" (tool names match binary names, and "dart" alone
// would shadow the SDK's own CLI shape), with the "dart" alias resolving
// to it.
//
// Check: "dart format --set-exit-if-changed -o none" exits 1 when
// changes are needed and writes nothing ("-o none" selects the null
// output). Dry-run maps to the same invocation (it never writes, and it
// conveys the changes-needed status). Arguments are forwarded verbatim;
// the optional "page_width"/"trailing_commas" settings live in the
// project's analysis_options.yaml `formatter:` section (language version
// ≥ 3.7) and are discovered by the tool itself, as is the
// "// dart format width=NNN" per-file marker.
func runDartFormat(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("dart", opts, []string{"format"})
	if err != nil || done {
		return code, err
	}

	switch {
	case opts.Check, opts.DryRun:
		// Fail if any file needs formatting; don't modify files.
		args := append([]string{"format", "--set-exit-if-changed", "-o", "none"}, forwarded...)
		return runTool(executable, args, false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"format"}, forwarded), false)
	}
}

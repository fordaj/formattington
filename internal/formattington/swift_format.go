package formattington

// swiftFormatExtensions are the file extensions handled by swift-format.
var swiftFormatExtensions = []string{".swift"}

// runSwiftFormat formats Swift files using swift-format (swiftlang/swift-format).
//
// Resolution prefers a standalone "swift-format" on PATH (newer
// standalone installs, e.g. from brew); when absent, the integration
// falls back to "swift format", the subcommand dispatcher included with
// Swift 6+ toolchains (Xcode 16+). Arguments are identical either way.
//
// Invocation is subcommand-shaped: "format" for in-place and "lint" with
// "--strict" for check (findings become errors; without --strict they are
// warnings and exit 0). "--ignore-unparsable-files" skips unparsable
// input instead of failing the whole batch. There is no diff mode, so
// --dry-run maps to the same --strict lint invocation. Arguments are
// forwarded verbatim; configuration is a ".swift-format" JSON file
// discovered by walking up from each file ("--configuration" overrides).
func runSwiftFormat(opts Options) (int, error) {
	executable, prefix, err := resolveSwiftFormat(opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, append(prefix, forwarded...), false)
	}

	switch {
	case opts.Check:
		// Fail if any findings exist; don't modify files.
		return runTool(executable, withArgs(append(prefix, "lint", "--strict", "--ignore-unparsable-files"), forwarded), false)
	case opts.DryRun:
		// Validate without modifying files.
		return runTool(executable, withArgs(append(prefix, "lint", "--strict", "--ignore-unparsable-files"), forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs(append(prefix, "-i", "--ignore-unparsable-files"), forwarded), false)
	}
}

// resolveSwiftFormat finds a standalone "swift-format" on PATH first; when
// absent, it falls back to "swift format" (Swift 6+ toolchains) and
// returns the "format" prefix to invoke.
func resolveSwiftFormat(verbose bool) (string, []string, error) {
	if executable, err := FindTool("swift-format", "", verbose); err == nil {
		return executable, nil, nil
	}
	if executable, err := FindTool("swift", "", verbose); err == nil {
		return executable, []string{"format"}, nil
	}
	// Neither is available; report against the standalone name.
	_, err := FindTool("swift-format", "", verbose)
	return "", nil, err
}

package formattington

import "strings"

// googleJavaFormatExtensions are the file extensions handled by
// google-java-format.
var googleJavaFormatExtensions = []string{".java"}

// runGoogleJavaFormat formats Java files using google-java-format.
//
// google-java-format v2.0.0 enables a non-whitespace "--cleanup" pass
// (redundant semicolons and the like) by default; Formattington passes
// "--cleanup=false" so diffs stay formatting-only, unless the caller
// forwards their own "--cleanup" value. The all-deps jar (requires a
// user-managed JDK >= 21 plus the six --add-exports flags) and
// --style=google|aosp are respected via forwarded args; there is no
// configuration file by design.
//
// "--dry-run --set-exit-if-changed" exits 1 when files would change and
// never writes; that is the check mapping (and the dry-run mapping —
// there is no patch output). "--replace" formats in place.
func runGoogleJavaFormat(opts Options) (int, error) {
	executable, err := FindTool("google-java-format", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	if !hasFlag(forwarded, "--cleanup") {
		// Whitespace-only formatting diffs by default.
		forwarded = withArgs([]string{"--cleanup=false"}, forwarded)
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		return runTool(executable, withArgs([]string{"--dry-run", "--set-exit-if-changed"}, forwarded), false)
	case opts.DryRun:
		// Validate without modifying files (same invocation; there is
		// no patch output).
		return runTool(executable, withArgs([]string{"--dry-run", "--set-exit-if-changed"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"--replace"}, forwarded), false)
	}
}

// hasFlag reports whether args already contain flag (either bare or as
// "--flag=value").
func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

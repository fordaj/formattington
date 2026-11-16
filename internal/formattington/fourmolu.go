package formattington

// fourmoluExtensions are the file extensions handled by fourmolu.
//
// Literate Haskell (.lhs) was not verified against the released binary
// (the GHC parser in Haddock mode rejects even trivial literate input),
// so only .hs is registered.
var fourmoluExtensions = []string{".hs"}

// fourmoluExitCheck means files need formatting; fourmolu --mode check
// prints a unified diff to stderr and exits 100.
const fourmoluExitCheck = 100

// runFourmolu formats Haskell files using Fourmolu (github.com/fourmolu/fourmolu).
//
// Check mode is "fourmolu --mode check": it prints a unified diff to
// stderr and exits 100 when files need formatting (0 clean; 102 other
// failures; 101 stdin/inplace conflict). Exit 100 is
// reported as ExitFormatRequired; all other codes pass through verbatim,
// so the stderr diff is never swallowed. There is no separate diff mode,
// so --dry-run maps to the same --mode check invocation. Default mode is
// "--mode inplace". Arguments are forwarded verbatim ("--indentation",
// "--config", ...); configuration is "fourmolu.yaml" discovered by walking
// up from each file, plus "--no-cabal" to ignore .cabal files'
// default-extensions.
func runFourmolu(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("fourmolu", opts, nil)
	if err != nil || done {
		return code, err
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		return remapFourmoluExit(runTool(executable, withArgs([]string{"--mode", "check"}, forwarded), false))
	case opts.DryRun:
		// Print a diff without modifying files (same invocation).
		return remapFourmoluExit(runTool(executable, withArgs([]string{"--mode", "check"}, forwarded), false))
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"--mode", "inplace"}, forwarded), false)
	}
}

// remapFourmoluExit maps fourmolu's exit-100 "formatting needed" to
// ExitFormatRequired; every other code passes through untouched.
func remapFourmoluExit(code int, err error) (int, error) {
	if err != nil {
		return ExitNotFound, err
	}
	if code == fourmoluExitCheck {
		return ExitFormatRequired, nil
	}
	return code, nil
}

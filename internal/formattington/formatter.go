package formattington

// Options carries the execution options for a single formatter run.
type Options struct {
	// Check reports formatting violations without modifying files.
	Check bool
	// DryRun validates without modifying files.
	DryRun bool
	// Args are the arguments forwarded verbatim to the underlying tool.
	Args []string
	// Verbose reports the resolved executable on stderr.
	Verbose bool
}

// Formatter runs one tool and returns its exit code. A non-nil error means
// the executable could not be resolved or started; the CLI reports it and
// exits 127.
type Formatter func(Options) (int, error)

// withArgs returns a copy of args with prefix prepended.
func withArgs(prefix []string, args []string) []string {
	out := make([]string, 0, len(prefix)+len(args))
	out = append(out, prefix...)
	out = append(out, args...)
	return out
}

// beginIntegration is the shared opening of every integration: it resolves
// the executable for toolName and copies the arguments to forward to it.
// prefix, when non-nil, is the subcommand token(s) the tool is invoked
// with (for example ["format"] for "dart format"); it is applied only on
// the informational path.
//
// When the forwarded arguments are an informational help/version
// passthrough, the tool is run immediately (with prefix applied) and
// (code, true) is reported; callers return code unchanged. Otherwise the
// executable and a fresh copy of the arguments (without prefix) are
// reported with done=false and code=ExitOK. A non-nil error means the
// executable could not be resolved (code is ExitNotFound).
func beginIntegration(toolName string, opts Options, prefix []string) (executable string, args []string, done bool, code int, err error) {
	executable, err = FindTool(toolName, "", opts.Verbose)
	if err != nil {
		return "", nil, false, ExitNotFound, err
	}
	args = append([]string{}, opts.Args...)
	if IsInformational(args) {
		runCode, runErr := runTool(executable, withArgs(prefix, args), false)
		if runErr != nil {
			return "", nil, false, ExitNotFound, runErr
		}
		return "", nil, true, runCode, nil
	}
	return executable, args, false, ExitOK, nil
}

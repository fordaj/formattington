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

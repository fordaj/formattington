package formattington

// taploExtensions are the file extensions handled by Taplo.
var taploExtensions = []string{".toml"}

// runTaplo formats TOML files using Taplo ("taplo fmt").
//
// Taplo respects project "taplo.toml" configuration (found automatically by
// Taplo); no "--config" flag is added unless the caller forwards one via
// Args. "--colors never" is forced so diff/log output is deterministic in
// captured output on every platform, unless the caller forwards their own
// "--colors" value.
func runTaplo(opts Options) (int, error) {
	executable, err := FindTool("taplo", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	arguments := []string{"fmt"}
	if !contains(forwarded, "--colors") {
		arguments = append(arguments, "--colors", "never")
	}
	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		arguments = append(arguments, "--check")
	case opts.DryRun:
		// Show diff without modifying files.
		arguments = append(arguments, "--diff")
	}
	// Default: taplo fmt writes in place without a flag.
	arguments = append(arguments, forwarded...)

	return runTool(executable, arguments, false)
}

package formattington

// forgeFmtExtensions are the file extensions handled by forge fmt.
var forgeFmtExtensions = []string{".sol"}

// runForgeFmt formats Solidity files using "forge fmt" (the standard
// Solidity toolchain formatter, from Foundry; install via foundryup,
// Homebrew, npm, or release tarballs).
//
// The binary is "forge", the subcommand is "fmt"; the Formattington tool
// name is "forge-fmt" with the "solidity" alias resolving to it.
//
// "forge fmt --check <files...>" exits 0 when files are formatted and 1
// when changes are required (a diff is printed by default; "-r"/"--raw"
// for raw output). --dry-run maps to the same --check invocation. The
// "[fmt]" table in foundry.toml is the only configuration source (line
// length, tab width, quote style, int_types, sort_imports, ignore, ...),
// discovered by the tool from the project root/git root; "--nearest"
// uses each file's nearest foundry.toml. Arguments are forwarded
// verbatim.
func runForgeFmt(opts Options) (int, error) {
	executable, err := FindTool("forge", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, withArgs([]string{"fmt"}, forwarded), false)
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		return runTool(executable, withArgs([]string{"fmt", "--check"}, forwarded), false)
	case opts.DryRun:
		// Validate without modifying files.
		return runTool(executable, withArgs([]string{"fmt", "--check"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"fmt"}, forwarded), false)
	}
}

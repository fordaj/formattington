package formattington

import "os"

// gnatformatExtensions are the file extensions handled by GNATformat.
// The ".gpr" project file is not formatted — it is the configuration.
var gnatformatExtensions = []string{".ads", ".adb"}

// runGnatformat formats Ada files using GNATformat (AdaCore/GNATformat),
// a port of the Prettier engine implementing the GNAT Coding Style.
//
// Project-driven model: formatting options come from the
// ".gpr" project file. When exactly one .gpr exists in the working
// directory, the tool auto-loads it (tool-native behavior; not
// second-guessed). When NO .gpr exists in the working directory, Formattington
// passes "--no-project" so standalone files are formatted with built-in
// defaults.
//
// Check mode is "--check": exits 1 when any file is not formatted (0
// otherwise). There is no built-in diff; "--pipe" prints the formatted
// source to stdout (pipe through "diff -u" yourself). --dry-run maps to
// the same --check invocation. Args verbatim; "--unparsing-configuration"
// selects a JSON configuration file.
func runGnatformat(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("gnatformat", opts, nil)
	if err != nil || done {
		return code, err
	}

	// Exactly one .gpr in cwd ⇒ the tool auto-loads it. None ⇒ standalone.
	projectArgs := []string{}
	if gprCount := countGprFiles("."); gprCount == 0 && !contains(forwarded, "--no-project") && !contains(forwarded, "-P") {
		projectArgs = append(projectArgs, "--no-project")
	}

	switch {
	case opts.Check:
		// Fail if any file is not formatted; don't modify files.
		return runTool(executable, withArgs(append(projectArgs, "--check"), forwarded), false)
	case opts.DryRun:
		// Validate without modifying files.
		return runTool(executable, withArgs(append(projectArgs, "--check"), forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs(projectArgs, forwarded), false)
	}
}

// countGprFiles counts the .gpr files in dir. Any OS error yields 0
// (standalone mode).
func countGprFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if pathSuffix(entry.Name()) == ".gpr" {
			count++
		}
	}
	return count
}

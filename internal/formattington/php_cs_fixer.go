package formattington

// phpCsFixerExitChanged is the exit code php-cs-fixer reports when files
// need to be fixed (for example in "check" mode). It is remapped to
// ExitFormatRequired; every other code is passed through untouched.
const phpCsFixerExitChanged = 8

// phpCsFixerExtensions are the file extensions handled by PHP-CS-Fixer.
var phpCsFixerExtensions = []string{".php"}

// runPhpCsFixer fixes PHP files using PHP-CS-Fixer.
//
// "php-cs-fixer check" is shorthand for "fix --dry-run" and exits 8 when
// files need fixing; that code is remapped to ExitFormatRequired. Every
// other non-zero exit code is passed through verbatim. PHP-CS-Fixer has no
// genuine check that other tools agree on, so --dry-run maps to the same
// non-modifying "check" invocation. Arguments are forwarded verbatim after
// the subcommand; configuration (".php-cs-fixer.php" /
// ".php-cs-fixer.dist.php") is discovered by the tool itself in the current
// directory, and can be overridden with "--config=path".
func runPhpCsFixer(opts Options) (int, error) {
	executable, err := FindTool("php-cs-fixer", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	if opts.Check || opts.DryRun {
		arguments := withArgs([]string{"check"}, forwarded)
		code, runErr := runTool(executable, arguments, false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code == phpCsFixerExitChanged {
			return ExitFormatRequired, nil
		}
		return code, nil
	}

	// Format in place.
	return runTool(executable, withArgs([]string{"fix"}, forwarded), false)
}

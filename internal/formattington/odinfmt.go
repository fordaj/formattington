package formattington

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// odinfmtExtensions are the file extensions handled by odinfmt.
var odinfmtExtensions = []string{".odin"}

// runOdinfmt formats Odin source files using odinfmt (lives in
// DanielGavin/ols, the Odin Language Server; distributed via Homebrew
// and nixpkgs — there is no Windows binary; install via
// "brew install odinfmt" or from ols source).
//
// "odinfmt -w" rewrites files in place; note it creates a
// "<file>_bk" backup before replacing the file (documented — users
// may want to gitignore "*.odin_bk").
//
// There is no check mode upstream. Formattington SYNTHESIZES one:
// "odinfmt <file>" prints the formatted source to stdout, and each
// output is byte-compared with the original — exit 1 when any file
// would change (the user tree is never modified). stdout mode cannot
// delimit multiple files, so synthesis runs one process per file.
// Arguments are forwarded verbatim; configuration is "odinfmt.json"
// discovered by walking up from each file ("--config" overrides).
func runOdinfmt(opts Options) (int, error) {
	executable, forwarded, done, code, err := beginIntegration("odinfmt", opts, nil)
	if err != nil || done {
		return code, err
	}

	files, flags := splitArgsAndFiles(forwarded, nil)

	if opts.Check || opts.DryRun {
		return odinfmtSynthesizedCheck(executable, files, flags)
	}

	// Format in place.
	args := append([]string{"-w"}, flags...)
	args = append(args, files...)
	return runTool(executable, args, false)
}

// odinfmtSynthesizedCheck implements check mode without touching the
// user's tree: "odinfmt <file>" is run per file and stdout is
// byte-compared with the original. Differing files are reported on
// stderr; exit 1 is returned when any differ. Tool errors pass through
// verbatim.
func odinfmtSynthesizedCheck(executable string, files, flags []string) (int, error) {
	if len(files) == 0 {
		// Suite-wide contract: the tool is still invoked once.
		return runTool(executable, append([]string{"--help"}, flags...), false)
	}

	needed := []string{}
	for _, file := range files {
		before, readErr := os.ReadFile(file)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "formattington: cannot read %s: %v\n", file, readErr)
			continue
		}
		args := append(append([]string{}, flags...), file)
		cmd := exec.Command(executable, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if runErr := cmd.Run(); runErr != nil {
			if isExitError(runErr) {
				code := cmd.ProcessState.ExitCode()
				if code != ExitOK {
					fmt.Fprint(os.Stderr, stderr.String())
					return code, nil // tool errors pass through verbatim
				}
			} else {
				return ExitNotFound, fmt.Errorf("could not run %s: %w", executable, runErr)
			}
		}
		if !bytes.Equal(stdout.Bytes(), before) {
			needed = append(needed, file)
		}
	}
	if len(needed) > 0 {
		for _, file := range needed {
			fmt.Fprintf(os.Stderr, "formattington: would reformat %s\n", file)
		}
		return ExitFormatRequired, nil
	}
	return ExitOK, nil
}

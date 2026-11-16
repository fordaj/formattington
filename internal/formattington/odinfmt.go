package formattington

import (
	"bytes"
	"errors"
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

// odinfmtFile formats one file and compares the tool's stdout with
// before. It reports diff when they differ, or code != ExitOK when the
// tool itself failed (its stderr is printed and its code returned for
// verbatim passthrough). A non-nil error means the tool could not be
// started.
func odinfmtFile(executable string, flags []string, file string, before []byte) (diff bool, code int, err error) {
	args := append(append([]string{}, flags...), file)
	cmd := exec.Command(executable, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		code, ok := toolExitCode(runErr)
		if !ok {
			return false, ExitNotFound, fmt.Errorf("could not run %s: %w", executable, runErr)
		}
		if code != ExitOK {
			fmt.Fprint(os.Stderr, stderr.String())
			return false, code, nil
		}
	}
	return !bytes.Equal(stdout.Bytes(), before), ExitOK, nil
}

// toolExitCode reports the exit code of a finished process; ok is false
// when err means the process never ran (failed to start).
func toolExitCode(err error) (int, bool) {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, false
	}
	return exitErr.ExitCode(), true
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
		diff, code, runErr := odinfmtFile(executable, flags, file, before)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code != ExitOK {
			return code, nil
		}
		if diff {
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

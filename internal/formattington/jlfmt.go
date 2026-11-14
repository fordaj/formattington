package formattington

import (
	"fmt"
	"os"
	"os/exec"
)

// jlfmtExtensions are the file extensions handled by jlfmt.
var jlfmtExtensions = []string{".jl"}

// runJlfmt formats Julia files using jlfmt (JuliaFormatter.jl's app).
//
// jlfmt is installed as a Julia app (Pkg >= 1.12):
//
//	julia -e 'using Pkg; Pkg.Apps.add("JuliaFormatter")'
//
// and put on PATH. There is no bundle — Julia is ~600 MB installed, so a
// bundled runtime was never on the table; PATH-only.
//
// Check mode is "jlfmt --check": exits 1 when any file needs formatting
// (0 clean, 2 on error; exit 2 passes through verbatim). "--diff" prints
// a unified diff computed with "git --no-index", so it requires git on
// PATH; when git is absent the diff invocation is replaced by --check
// with a clear note on stderr (a degraded but working validation).
// Arguments are forwarded verbatim; ".JuliaFormatter.toml" is discovered
// by walking up from each file (synergy with Taplo for TOML users).
func runJlfmt(opts Options) (int, error) {
	executable, err := FindTool("jlfmt", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	switch {
	case opts.Check:
		// Fail if any file needs formatting; don't modify files.
		code, runErr := runTool(executable, withArgs([]string{"--check"}, forwarded), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code == ExitFormatRequired {
			return ExitFormatRequired, nil
		}
		return code, nil
	case opts.DryRun:
		// Diff requires git (jlfmt computes it with git --no-index).
		if _, gitErr := exec.LookPath("git"); gitErr != nil {
			fmt.Fprintln(os.Stderr,
				"formattington: jlfmt --diff requires git on PATH; running --check instead")
			return runTool(executable, withArgs([]string{"--check"}, forwarded), false)
		}
		return runTool(executable, withArgs([]string{"--diff"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"--inplace"}, forwarded), false)
	}
}

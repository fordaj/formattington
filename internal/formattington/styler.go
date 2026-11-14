package formattington

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// stylerExtensions are the file extensions handled by styler.
var stylerExtensions = []string{".R", ".r"}

// runStyler formats R files using the styler package via Rscript.
// (styler is an R package with no standalone executable; installing R is
// required. PATH-only — bundling R is out of scope.)
//
// The R expression is constructed inside the integration from the file
// list; user arguments are never spliced into it (injection risk), so a
// forwarded flag is a usage error — do advanced customization through a
// wrapper script or an R-level options() call instead. Non-flag
// arguments are taken as the file list, exactly like every other tool.
//
// Check mode uses styler::style_file(..., dry = "fail"), which does not
// write and errors out when any file would be modified — the non-zero
// exit is reported as ExitFormatRequired. Dry-run maps to the same
// invocation (it reports which files would change). The default uses
// dry = "off" (rewrites files in place). Paths are quoted for R with
// strconv.Quote, which matches R's double-quoted string escapes for the
// paths Formattington handles.
func runStyler(opts Options) (int, error) {
	executable, err := FindTool("Rscript", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	files := []string{}
	for _, arg := range forwarded {
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr,
				"formattington: styler: forwarded flag %q is not supported; use a wrapper script or R-level options() for advanced customization\n",
				arg)
			return ExitUsage, nil
		}
		files = append(files, arg)
	}
	expression := "invisible(NULL)"
	if len(files) > 0 {
		quoted := make([]string, 0, len(files))
		for _, file := range files {
			quoted = append(quoted, strconv.Quote(file))
		}
		dry := "off"
		if opts.Check || opts.DryRun {
			dry = "fail"
		}
		expression = "styler::style_file(c(" + strings.Join(quoted, ", ") + "), dry = \"" + dry + "\")"
	}

	code, runErr := runTool(executable, []string{"-e", expression}, false)
	if runErr != nil {
		return ExitNotFound, runErr
	}
	if (opts.Check || opts.DryRun) && code != ExitOK {
		// styler errors abort with dryError when a file would be
		// modified; treat any non-zero exit as "changes required".
		return ExitFormatRequired, nil
	}
	return code, nil
}

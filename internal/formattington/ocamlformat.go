package formattington

// ocamlformatExtensions are the file extensions handled by ocamlformat.
var ocamlformatExtensions = []string{".ml", ".mli"}

// runOcamlformat formats OCaml files using ocamlformat
// (github.com/ocaml-ppx/ocamlformat, via "opam install ocamlformat").
//
// There is no bundle: ocamlformat ships only as a source tarball built
// through opam against the user's switch — no native binaries exist to
// redistribute.
//
// Check mode is "ocamlformat --check": exit 1 when files need formatting.
// The same exit 1 is used for errors (indistinguishable — stderr
// disambiguates), so it is reported as ExitFormatRequired either way.
// There is no diff mode; --dry-run maps to the same --check invocation.
// Arguments are forwarded verbatim ("--profile", "--enable-outside-detected-project",
// ...). The mandatory ".ocamlformat" "version = <installed release>" pin
// is the tool's own enforcement: a mismatched pin refuses to run —
// reconcile the pin with the installed release (or run "dune build" which
// pins it).
func runOcamlformat(opts Options) (int, error) {
	executable, err := FindTool("ocamlformat", "", opts.Verbose)
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
		return runTool(executable, withArgs([]string{"--check"}, forwarded), false)
	case opts.DryRun:
		// No diff mode exists; validate without modifying files.
		return runTool(executable, withArgs([]string{"--check"}, forwarded), false)
	default:
		// Format in place.
		return runTool(executable, withArgs([]string{"--inplace"}, forwarded), false)
	}
}

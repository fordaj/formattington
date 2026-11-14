package formattington

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// mojoFormatExtensions are the file extensions handled by mojo format.
var mojoFormatExtensions = []string{".mojo"}

// mojoValuedFlags are the mojo format flags that take a value.
var mojoValuedFlags = map[string]bool{
	"-l":            true,
	"--line-length": true,
}

// runMojoFormat formats Mojo files using "mojo format" (ships with the
// Mojo toolchain).
//
// "mojo format" modifies files in place, period — there is no check mode
// and no diff mode upstream. Formattington SYNTHESIZES check mode the
// safe way: each input file is copied into a temporary directory,
// "mojo format" is run on the COPIES, the copies are byte-compared with
// the originals, and exit 1 is reported when any file differs (the user
// tree is never modified). Non-zero exits from the tool itself pass
// through untouched. The same synthesis backs --dry-run.
//
// Platform support: Mojo runs only on Linux (glibc ≥ 2.34, x86-64-v3 /
// Neoverse N1+) and macOS Apple silicon; Windows users must use WSL.
// Arguments are forwarded verbatim ("-l 100" / "-q"), applied to the
// copies in check mode just as they would be in place with default mode.
// There is no configuration file, and none is invented.
func runMojoFormat(opts Options) (int, error) {
	executable, err := FindTool("mojo", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, withArgs([]string{"format"}, forwarded), false)
	}

	files, flags := splitArgsAndFiles(forwarded, mojoValuedFlags)

	if opts.Check || opts.DryRun {
		return mojoFormatSynthesizedCheck(executable, files, flags)
	}

	// Format in place.
	args := append([]string{"format"}, flags...)
	args = append(args, files...)
	return runTool(executable, args, false)
}

// mojoFormatSynthesizedCheck implements check mode without touching the
// user's tree: files are copied to a temp dir, formatted in place there,
// and byte-compared. Differing files are reported on stderr; exit 1 is
// returned when any differ. Tool errors pass through verbatim.
func mojoFormatSynthesizedCheck(executable string, files, flags []string) (int, error) {
	if len(files) == 0 {
		// Suite-wide contract: the tool is still invoked once. Use
		// --help so an interactive mojo never blocks on stdin.
		return runTool(executable, []string{"format", "--help"}, false)
	}

	tmp, err := os.MkdirTemp("", "formattington-mojo-*")
	if err != nil {
		return ExitNotFound, err
	}
	defer os.RemoveAll(tmp)

	copies := make(map[string]string, len(files)) // temp path -> original path
	before := make(map[string][]byte, len(files))
	for index, file := range files {
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "formattington: cannot read %s: %v\n", file, readErr)
			continue
		}
		dst := filepath.Join(tmp, fmt.Sprintf("%06d.mojo", index))
		if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
			return ExitNotFound, writeErr
		}
		copies[dst] = file
		before[file] = data
	}

	args := append([]string{"format"}, flags...)
	for dst := range copies {
		args = append(args, dst)
	}
	code, runErr := runTool(executable, args, false)
	if runErr != nil {
		return ExitNotFound, runErr
	}
	if code != ExitOK {
		return code, nil // tool errors pass through verbatim
	}

	needed := []string{}
	for dst, original := range copies {
		after, readErr := os.ReadFile(dst)
		if readErr != nil {
			return ExitNotFound, readErr
		}
		if !bytes.Equal(after, before[original]) {
			needed = append(needed, original)
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

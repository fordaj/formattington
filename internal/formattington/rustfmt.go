package formattington

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// rustfmtExtensions are the file extensions handled by rustfmt.
var rustfmtExtensions = []string{".rs"}

// runRustfmt formats Rust files using rustfmt (rustup component add
// rustfmt).
//
// Edition handling: rustfmt's default edition is 2015, silently wrong
// for most modern crates. When the caller does NOT pass --edition, the
// integration walks up from each file for the nearest Cargo.toml and
// uses its [package] edition; a member whose edition is inherited from
// a virtual workspace root uses the nearest [workspace.package]
// edition. Files are grouped by edition and batched accordingly;
// sibling files with different editions are formatted with their own
// --edition. Files with no discoverable Cargo.toml anywhere up the tree
// are formatted with rustfmt's own default (and the README explains the
// flag passthrough to set an edition explicitly).
//
// "--check" exits 1 when files differ (also used for parse errors —
// indistinguishable; stderr disambiguates). "--emit diff" prints a diff.
// Args are forwarded verbatim; a caller-provided --edition wins over the
// detected one.
func runRustfmt(opts Options) (int, error) {
	executable, err := FindTool("rustfmt", "", opts.Verbose)
	if err != nil {
		return ExitNotFound, err
	}
	forwarded := append([]string{}, opts.Args...)

	// Preserve native informational commands without adding format flags.
	if IsInformational(forwarded) {
		return runTool(executable, forwarded, false)
	}

	userHasEdition := hasFlag(forwarded, "--edition")
	files, flags := splitArgsAndFiles(forwarded)

	// Partition files by detected edition (empty = tool default).
	groups := map[string][]string{}
	for _, file := range files {
		edition := ""
		if !userHasEdition {
			edition = detectEdition(file)
		}
		groups[edition] = append(groups[edition], file)
	}

	// Deterministic batch order: editions sorted, so the aggregated
	// exit code is stable for multi-edition runs.
	editions := make([]string, 0, len(groups))
	for edition := range groups {
		editions = append(editions, edition)
	}
	sort.Strings(editions)

	exitCode := ExitOK
	if len(groups) == 0 {
		// No files (nothing discovered and no user args): the
		// suite-wide contract is that the tool is still invoked.
		args := []string{}
		switch {
		case opts.Check:
			args = append(args, "--check")
		case opts.DryRun:
			args = append(args, "--emit", "diff")
		}
		code, runErr := runTool(executable, withArgs(args, flags), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		return code, nil
	}
	for _, edition := range editions {
		group := groups[edition]
		args := []string{}
		if edition != "" {
			args = append(args, "--edition", edition)
		}
		switch {
		case opts.Check:
			args = append(args, "--check")
		case opts.DryRun:
			args = append(args, "--emit", "diff")
		}
		code, runErr := runTool(executable, withArgs(args, append(flags, group...)), false)
		if runErr != nil {
			return ExitNotFound, runErr
		}
		if code != ExitOK {
			exitCode = code
		}
	}
	return exitCode, nil
}

// rustfmtValuedFlags are rustfmt flags that consume the following
// argument as their value; they must not be mistaken for a file path.
var rustfmtValuedFlags = map[string]bool{
	"--emit":         true,
	"--edition":      true,
	"--config-path":  true,
	"--config":       true,
	"--color":        true,
	"--error-format": true,
}

// splitArgsAndFiles separates non-flag arguments (treated as files)
// from everything else, preserving order. Known rustfmt flags that take
// a value consume the next argument as part of the flag.
func splitArgsAndFiles(args []string) (files []string, rest []string) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if strings.HasPrefix(arg, "-") {
			rest = append(rest, arg)
			if rustfmtValuedFlags[arg] && index+1 < len(args) {
				rest = append(rest, args[index+1])
				index++
			}
			continue
		}
		files = append(files, arg)
	}
	return files, rest
}

// detectEdition walks up from the file's directory looking for a
// Cargo.toml with a [package] edition; a member without its own edition
// uses the nearest [workspace.package] edition found further up. Empty
// string means "no edition discoverable" (rustfmt's default applies).
func detectEdition(file string) string {
	dir := filepath.Dir(file)
	for {
		cargo := filepath.Join(dir, "Cargo.toml")
		packageEdition, workspaceEdition := readCargoEditions(cargo)
		if packageEdition != "" {
			return packageEdition
		}
		if workspaceEdition != "" {
			// Nearest [workspace.package] wins (covers edition
			// inheritance in virtual/workspace manifests).
			return workspaceEdition
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// readCargoEditions returns the [package] edition and
// [workspace.package] edition from a Cargo.toml, if any.
func readCargoEditions(path string) (string, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	text := string(data)
	// Walk section headers, tracking the current section.
	header := regexp.MustCompile(`(?m)^\s*\[([^\]]+)\]`)
	sections := header.FindAllStringSubmatchIndex(text, -1)
	editionRe := regexp.MustCompile(`(?m)^\s*edition\s*=\s*"(20[0-9]+)"`)
	packageEdition, workspaceEdition := "", ""
	for i, match := range sections {
		name := strings.ToLower(strings.TrimSpace(text[match[2]:match[3]]))
		content := text[match[1]:]
		if i+1 < len(sections) {
			content = text[match[1]:sections[i+1][0]]
		}
		if ed := editionRe.FindStringSubmatch(content); ed != nil {
			switch name {
			case "package":
				packageEdition = ed[1]
			case "workspace.package":
				workspaceEdition = ed[1]
			}
		}
	}
	return packageEdition, workspaceEdition
}

package formattington

import (
	"bufio"
	"bytes"
	_ "embed" // required by the //go:embed directive
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// initTemplate is the full catalogue of formatter invocations the "init"
// command draws from, one command per registered tool in registration
// order. It is embedded so a downloaded binary can generate a project's
// .formattington.json without the repository checkout. The file lives in
// the source tree as internal/formattington/init.json.
//
//go:embed init.json
var initTemplate []byte

// initScan holds the outcome of examining the current directory: the
// number of files that contributed a recognized suffix and the per-suffix
// counts.
type initScan struct {
	files    int
	suffixes map[string]int
}

// record counts one file with the given suffix.
func (s *initScan) record(suffix string) {
	s.files++
	if s.suffixes == nil {
		s.suffixes = make(map[string]int)
	}
	s.suffixes[suffix]++
}

// merge folds other into s.
func (s *initScan) merge(other initScan) {
	s.files += other.files
	if s.suffixes == nil {
		s.suffixes = make(map[string]int, len(other.suffixes))
	}
	for suffix, count := range other.suffixes {
		s.suffixes[suffix] += count
	}
}

// runInit implements "formattington init": it examines the current
// directory, reports what it found, and writes a .formattington.json whose
// commands are the template invocations for the file types present. It
// returns the process exit code.
func runInit(verbose bool) int {
	if _, err := os.Stat(configFilename); err == nil {
		if !stdinIsTerminal() {
			fmt.Fprintf(os.Stderr, "formattington: %s already exists; not overwriting\n", configFilename)
			return ExitUsage
		}
		if !confirmOverwrite(os.Stdin, os.Stdout, configFilename) {
			fmt.Println("Keeping existing .formattington.json.")
			return ExitOK
		}
	}

	fmt.Print("Examining current directory... ")
	scanned, err := scanCurrentDirectory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nformattington: %v\n", err)
		return ExitUsage
	}
	fmt.Printf("found %d files across %d file extensions.\n", scanned.files, len(scanned.suffixes))

	if verbose {
		for _, ext := range sortedKeys(scanned.suffixes) {
			fmt.Fprintf(os.Stderr, "formattington: %s: %d files\n", ext, scanned.suffixes[ext])
		}
	}

	template, err := initTemplateCommands()
	if err != nil {
		// The embedded template is validated by the test suite; a failure
		// here is a build/packaging bug, not user error.
		fmt.Fprintf(os.Stderr, "formattington: %v\n", err)
		return ExitUsage
	}
	commands := selectInitCommands(template, scanned.suffixes)
	data, err := renderInitConfig(commands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "formattington: could not render .formattington.json: %v\n", err)
		return ExitUsage
	}

	if err := os.WriteFile(configFilename, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "formattington: could not write %s: %v\n", configFilename, err)
		return ExitUsage
	}

	fmt.Printf("Wrote %s with %d command(s):\n", configFilename, len(commands))
	for _, command := range commands {
		fmt.Printf("  %s\n", strings.Join(command, " "))
	}
	if len(commands) == 0 {
		fmt.Println("  (no recognized file types found; 'commands' is empty and 'formattington all' will run nothing)")
	}
	return ExitOK
}

// scanCurrentDirectory examines the working directory and returns the
// aggregate scan. Tracked files are preferred: inside a Git repository
// only files Git knows about (tracked files, plus untracked files that are
// not excluded by .gitignore) are examined, submodules included, and a
// repository encountered during the scan is examined from its own tracked
// perspective. Otherwise every file is examined. Hidden directories and
// the generated/vendored/package-manager directory names are always
// skipped.
func scanCurrentDirectory() (initScan, error) {
	root, err := os.Getwd()
	if err != nil {
		return initScan{}, err
	}
	if dirHasGit(root) {
		if !gitAvailable() {
			// A .git marker exists but Git cannot be run; fall back to
			// examining all files rather than reporting an empty project.
			fmt.Fprintln(os.Stderr, "formattington: git is not available; scanning all files")
			return walkFiles(root), nil
		}
		return scanRepo(root), nil
	}
	return walkFiles(root), nil
}

// scanRepo scans a directory that is itself a Git repository (the ".git"
// marker exists or Git was invoked successfully). It prefers tracked
// files (including submodule tracked files via --recurse-submodules) and
// additionally counts untracked files that are not excluded by
// .gitignore, recursing into untracked subdirectories and nested
// repositories with a plain walk.
func scanRepo(root string) initScan {
	var scanned initScan
	tracked, err := scanGitList(root, true)
	if err != nil {
		// A corrupt repository or a git invocation failure must not
		// silently produce an empty config; fall back to a plain walk.
		fmt.Fprintf(os.Stderr, "formattington: could not list tracked files with git (%v); scanning all files\n", err)
		return walkFiles(root)
	}
	scanned.merge(tracked)
	scanned.merge(scanUntracked(root))
	scanned.merge(scanSubmoduleUntracked(root))
	return scanned
}

// scanGitList runs git ls-files --cached (optionally recursing into
// submodules) in root and counts the suffixes of the listed files that
// still exist on disk and are not under an ignored directory.
func scanGitList(root string, recurse bool) (initScan, error) {
	args := []string{"--cached", "-z"}
	if recurse {
		args = append(args, "--recurse-submodules")
	}
	entries, err := gitNulList(root, append([]string{"ls-files"}, args...)...)
	if err != nil {
		return initScan{}, err
	}
	var scanned initScan
	scanned.suffixes = make(map[string]int, len(entries))
	for _, rel := range entries {
		if suffix, ok := countableGitSuffix(root, rel); ok {
			scanned.record(suffix)
		}
	}
	return scanned, nil
}

// scanUntracked counts the untracked, non-ignored files of the repository
// at root. Git collapses untracked directories into a single "dir/" entry;
// entries for directories are examined with their own scan (a nested
// repository is scanned from its tracked perspective, any other directory
// is walked), so no untracked source tree is silently skipped.
func scanUntracked(root string) initScan {
	var scanned initScan
	scanned.suffixes = make(map[string]int)
	entries, err := gitNulList(root, "ls-files", "-z", "--others", "--directory", "--exclude-standard")
	if err != nil {
		return scanned
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry, "/") {
			sub := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(entry, "/")))
			if dirHasGit(sub) && gitAvailable() {
				scanned.merge(scanRepo(sub))
			} else {
				scanned.merge(walkFiles(sub))
			}
			continue
		}
		if suffix, ok := countableGitSuffix(root, entry); ok {
			scanned.record(suffix)
		}
	}
	return scanned
}

// scanSubmoduleUntracked counts untracked, non-ignored files inside each
// submodule of the repository at root (the parent repository reports
// submodule contents as a gitlink, so its own untracked listing cannot
// see them).
func scanSubmoduleUntracked(root string) initScan {
	var scanned initScan
	scanned.suffixes = make(map[string]int)
	for _, rel := range gitSubmodulePaths(root) {
		sub := filepath.Join(root, filepath.FromSlash(rel))
		if info, err := os.Stat(sub); err != nil || !info.IsDir() {
			// An uninitialized submodule has no files to examine.
			continue
		}
		scanned.merge(scanUntracked(sub))
	}
	return scanned
}

// gitSubmodulePaths returns the repo-relative paths of the gitlinks (mode
// 160000) in the index of the repository at dir.
func gitSubmodulePaths(dir string) []string {
	entries, err := gitNulList(dir, "ls-files", "-z", "--stage")
	if err != nil {
		return nil
	}
	var submodules []string
	for _, entry := range entries {
		tab := strings.IndexByte(entry, '\t')
		if tab < 0 {
			continue
		}
		fields := strings.Fields(entry[:tab])
		if len(fields) == 0 || fields[0] != "160000" {
			continue
		}
		submodules = append(submodules, entry[tab+1:])
	}
	return submodules
}

// walkFiles examines dir and its subtree, with no Git information: every
// regular file is counted and a directory containing a Git marker is
// scanned from its tracked perspective instead of walked. Hidden
// directories and the ignored directory names are skipped.
func walkFiles(root string) initScan {
	var scanned initScan
	scanned.suffixes = make(map[string]int)
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if skip := scanWalkEntry(root, path, entry, &scanned); skip {
			return filepath.SkipDir
		}
		return nil
	})
	return scanned
}

// scanWalkEntry examines one entry of the non-git walk and merges its
// findings into scanned. It reports whether the walk should skip the
// entry's subtree (an ignored directory or a directory examined from its
// tracked Git perspective instead).
func scanWalkEntry(root, path string, entry fs.DirEntry, scanned *initScan) bool {
	if entry.IsDir() {
		if path == root {
			return false
		}
		if isIgnoredDirName(entry.Name()) {
			return true
		}
		if dirHasGit(path) && gitAvailable() {
			scanned.merge(scanRepo(path))
			return true
		}
		return false
	}
	if !entry.Type().IsRegular() {
		return false
	}
	suffix, ok := countableGitSuffix(root, mustRelative(root, path))
	if !ok {
		return false
	}
	scanned.record(suffix)
	return false
}

// countableGitSuffix reports whether the Git- or walk-reported path rel
// (relative to root) contributes a suffix to the scan: every directory
// component must be an included name (see isIgnoredDirName) and the file
// must exist and be a regular file.
func countableGitSuffix(root, rel string) (string, bool) {
	native := filepath.FromSlash(rel)
	dir := filepath.Dir(native)
	for _, segment := range strings.Split(dir, string(os.PathSeparator)) {
		if segment == "." || segment == "" {
			continue
		}
		if isIgnoredDirName(segment) {
			return "", false
		}
	}
	suffix := pathSuffix(native)
	if suffix == "" {
		return "", false
	}
	info, err := os.Stat(filepath.Join(root, native))
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return suffix, true
}

// mustRelative returns path relative to root, or path itself on failure.
func mustRelative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// dirHasGit reports whether dir looks like a repository root: it contains
// a ".git" directory (normal repositories) or file (linked worktrees and
// submodules).
func dirHasGit(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// gitAvailable reports whether a git executable is on PATH.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// gitNulList runs git in dir with the given arguments and returns the
// NUL-separated output as a slice of entries.
func gitNulList(dir string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(out, []byte{0})
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		entries = append(entries, string(part))
	}
	return entries, nil
}

// initTemplateCommands parses and validates the embedded init template
// with the same validator the CLI applies to user configuration files.
func initTemplateCommands() ([][]string, error) {
	var parsed any
	if err := json.Unmarshal(initTemplate, &parsed); err != nil {
		return nil, fmt.Errorf("invalid embedded init template: %v", err)
	}
	commands, err := validateConfig(parsed, "init.json")
	if err != nil {
		return nil, err
	}
	return commands, nil
}

// selectInitCommands returns the template commands whose tool handles at
// least one detected suffix, preserving template order (the registration
// order) and the template's per-command arguments verbatim.
func selectInitCommands(template [][]string, suffixes map[string]int) [][]string {
	extensionsOf := make(map[string][]string, len(registry))
	for _, tool := range registry {
		extensionsOf[tool.Name] = tool.Extensions
	}
	var selected [][]string
	for _, command := range template {
		matched := false
		for _, extension := range extensionsOf[command[0]] {
			if suffixes[strings.ToLower(extension)] > 0 {
				matched = true
				break
			}
		}
		if matched {
			selected = append(selected, command)
		}
	}
	return selected
}

// renderInitConfig serializes commands as the JSON document
// .formattington.json holds. The commands are printed one per line (the
// shape of a handwritten config) with the same key order as a compacted
// json.Marshal of each entry, so two init runs produce identical bytes.
func renderInitConfig(commands [][]string) ([]byte, error) {
	var buf bytes.Buffer
	if len(commands) == 0 {
		buf.WriteString("{\n  \"commands\": []\n}\n")
		return buf.Bytes(), nil
	}
	buf.WriteString("{\n  \"commands\": [\n")
	for index, command := range commands {
		parts := make([]string, 0, len(command))
		for _, argument := range command {
			argumentJSON, err := json.Marshal(argument)
			if err != nil {
				return nil, err
			}
			parts = append(parts, string(argumentJSON))
		}
		suffix := ","
		if index == len(commands)-1 {
			suffix = ""
		}
		fmt.Fprintf(&buf, "    [%s]%s\n", strings.Join(parts, ", "), suffix)
	}
	buf.WriteString("  ]\n}\n")
	return buf.Bytes(), nil
}

// confirmOverwrite asks the user whether to overwrite path and reports
// the answer. Only an explicit "y" or "yes" (case-insensitive) confirms.
func confirmOverwrite(r io.Reader, out io.Writer, path string) bool {
	fmt.Fprintf(out, "formattington: %s already exists. Overwrite? [y/N] ", path)
	answer, err := readLine(r)
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	return false
}

// stdinIsTerminal reports whether standard input is an interactive terminal.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// readLine reads one line (without the trailing newline) from r.
func readLine(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	return strings.TrimSuffix(line, "\n"), err
}

// sortedKeys returns the keys of suffixes in ascending order.
func sortedKeys(suffixes map[string]int) []string {
	keys := make([]string, 0, len(suffixes))
	for key := range suffixes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

package formattington

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// gitCommand runs git in dir for test setup, failing the test on error.
func gitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available on this machine")
	}
}

// initGitRepo initializes a repository with one commit containing the
// given name/content pairs.
func initGitRepo(t *testing.T, root string, files map[string]string) {
	t.Helper()
	requireGit(t)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	gitCommand(t, root, "init", "-q", "-b", "main")
	for name, content := range files {
		writeFile(t, filepath.Join(root, name), content)
	}
	gitCommand(t, root, "add", "-A")
	gitCommand(t, root,
		"-c", "user.email=test@example.com",
		"-c", "user.name=test",
		"commit", "-qm", "init")
}

// suffixMap returns a single-element readable projection of a scan for
// failure messages.
func suffixMap(scanned initScan) map[string]int {
	return scanned.suffixes
}

func TestWalkScansAllFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.sh"), "echo hi\n")
	writeFile(t, filepath.Join(root, "b.md"), "# hi\n")
	writeFile(t, filepath.Join(root, "sub", "c.py"), "x = 1\n")
	writeFile(t, filepath.Join(root, "noext"), "no extension\n")
	writeFile(t, filepath.Join(root, "trailingdot."), "no suffix\n")

	scanned := walkFiles(root)
	want := map[string]int{".sh": 1, ".md": 1, ".py": 1}
	if !reflect.DeepEqual(suffixMap(scanned), want) {
		t.Errorf("suffixes = %v, want %v", scanned.suffixes, want)
	}
	if scanned.files != 3 {
		t.Errorf("files = %d, want 3", scanned.files)
	}
}

func TestWalkSkipsGeneratedAndVendoredDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep.md"), "# keep\n")
	for _, dir := range []string{
		"node_modules", "vendor", "venv", ".venv", "dist", "build",
		"__pycache__", "tools", "formatters", ".hidden", ".git-config-dir",
	} {
		writeFile(t, filepath.Join(root, dir, "skip.py"), "x = 1\n")
	}

	scanned := walkFiles(root)
	want := map[string]int{".md": 1}
	if !reflect.DeepEqual(suffixMap(scanned), want) {
		t.Errorf("suffixes = %v, want %v", scanned.suffixes, want)
	}
}

func TestGitScanPrefersTrackedAndIgnoresUntrackedIgnored(t *testing.T) {
	root := t.TempDir()
	// node_modules/forced.py is deliberately committed (with -f semantics
	// by writing before commit) to prove the filter applies to tracked
	// files too.
	initGitRepo(t, root, map[string]string{
		"a.py":                   "x = 1\n",
		"b.rs":                   "fn main() {}\n",
		"node_modules/forced.py": "x = 1\n",
		"deleted.go":             "package p\n",
	})

	// A tracked file deleted from the working tree must not be counted.
	if err := os.Remove(filepath.Join(root, "deleted.go")); err != nil {
		t.Fatal(err)
	}

	// An untracked, non-ignored file counts; an ignored one does not.
	writeFile(t, filepath.Join(root, "untracked.sh"), "echo hi\n")
	writeFile(t, filepath.Join(root, "ignored.zzz"), "ignored\n")
	writeFile(t, filepath.Join(root, "keep.zig"), "var x: i32 = 1;\n")
	writeFile(t, filepath.Join(root, ".gitignore"), "ignored.zzz\nkeep.zig\n")
	// Note: .gitignore is plain text, so this marker has no suffix that
	// maps onto a registered tool.

	t.Chdir(root)
	scanned, err := scanCurrentDirectory()
	if err != nil {
		t.Fatalf("scanCurrentDirectory: %v", err)
	}

	for suffix := range scanned.suffixes {
		switch suffix {
		case ".py", ".rs", ".sh":
		default:
			t.Errorf("unexpected suffix %q in scan %v", suffix, scanned.suffixes)
		}
	}
	if scanned.suffixes[".py"] != 1 {
		t.Errorf(".py = %d, want 1 (forced tracked node_modules file must be excluded)", scanned.suffixes[".py"])
	}
	if scanned.suffixes[".rs"] != 1 {
		t.Errorf(".rs = %d, want 1", scanned.suffixes[".rs"])
	}
	if scanned.suffixes[".sh"] != 1 {
		t.Errorf(".sh = %d, want 1 (untracked, non-ignored file)", scanned.suffixes[".sh"])
	}
	if scanned.suffixes[".go"] != 0 {
		t.Errorf(".go = %d, want 0 (deleted tracked file must not count)", scanned.suffixes[".go"])
	}
	if scanned.suffixes[".zig"] != 0 {
		t.Errorf(".zig = %d, want 0 (gitignore-excluded file must not count)", scanned.suffixes[".zig"])
	}
}

func TestGitScanCountsSubmoduleTrackedFiles(t *testing.T) {
	requireGit(t)

	lib := t.TempDir()
	initGitRepo(t, lib, map[string]string{"lib.zig": "var x: i32 = 1;\n"})

	root := t.TempDir()
	initGitRepo(t, root, map[string]string{"app.py": "x = 1\n"})
	gitCommand(t, root, "-c", "protocol.file.allow=always", "submodule", "add", "-q", lib, "sub")
	gitCommand(t, root,
		"-c", "user.email=test@example.com",
		"-c", "user.name=test",
		"commit", "-qm", "add submodule")

	// An untracked file inside the submodule must also be counted.
	writeFile(t, filepath.Join(root, "sub", "untracked.gleam"), "pub fn main() {}\n")

	t.Chdir(root)
	scanned, err := scanCurrentDirectory()
	if err != nil {
		t.Fatalf("scanCurrentDirectory: %v", err)
	}

	if scanned.suffixes[".py"] != 1 {
		t.Errorf(".py = %d, want 1", scanned.suffixes[".py"])
	}
	if scanned.suffixes[".zig"] != 1 {
		t.Errorf(".zig = %d, want 1 (submodule tracked file)", scanned.suffixes[".zig"])
	}
	if scanned.suffixes[".gleam"] != 1 {
		t.Errorf(".gleam = %d, want 1 (untracked file inside submodule)", scanned.suffixes[".gleam"])
	}
}

func TestNestedRepositoryScannedFromTrackedPerspective(t *testing.T) {
	requireGit(t)

	// A repository that is not a (sub)module of the parent: examined from
	// its own tracked files, not walked file by file.
	nested := filepath.Join(t.TempDir(), "nested")
	initGitRepo(t, nested, map[string]string{
		"lib.rs":            "fn main() {}\n",
		"node_modules/x.py": "x = 1\n",
	})

	parent := t.TempDir()
	initGitRepo(t, parent, map[string]string{"app.sh": "echo hi\n"})
	if err := os.Rename(nested, filepath.Join(parent, "nested")); err != nil {
		t.Fatal(err)
	}

	t.Chdir(parent)
	scanned, err := scanCurrentDirectory()
	if err != nil {
		t.Fatalf("scanCurrentDirectory: %v", err)
	}

	if scanned.suffixes[".sh"] != 1 {
		t.Errorf(".sh = %d, want 1", scanned.suffixes[".sh"])
	}
	if scanned.suffixes[".rs"] != 1 {
		t.Errorf(".rs = %d, want 1 (nested repo tracked file)", scanned.suffixes[".rs"])
	}
	if scanned.suffixes[".py"] != 0 {
		t.Errorf(".py = %d, want 0 (nested repo's ignored file must not count)", scanned.suffixes[".py"])
	}
}

func TestNestedRepositoryInNonRepoDirectory(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app.py"), "x = 1\n")
	nested := filepath.Join(root, "nested")
	initGitRepo(t, nested, map[string]string{"lib.rs": "fn main() {}\n"})

	t.Chdir(root)
	scanned, err := scanCurrentDirectory()
	if err != nil {
		t.Fatalf("scanCurrentDirectory: %v", err)
	}

	if scanned.suffixes[".py"] != 1 {
		t.Errorf(".py = %d, want 1", scanned.suffixes[".py"])
	}
	if scanned.suffixes[".rs"] != 1 {
		t.Errorf(".rs = %d, want 1 (repository encountered during the walk)", scanned.suffixes[".rs"])
	}
}

func TestRenderInitConfig(t *testing.T) {
	commands := [][]string{{"prettier"}, {"shfmt", "-i", "2"}}
	want := "{\n  \"commands\": [\n    [\"prettier\"],\n    [\"shfmt\", \"-i\", \"2\"]\n  ]\n}\n"
	got, err := renderInitConfig(commands)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("renderInitConfig = %q, want %q", got, want)
	}

	got, err = renderInitConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"commands\": []\n}\n"; string(got) != want {
		t.Errorf("renderInitConfig(nil) = %q, want %q", got, want)
	}
}

func TestSelectInitCommandsFiltersAndPreservesOrder(t *testing.T) {
	template, err := initTemplateCommands()
	if err != nil {
		t.Fatalf("initTemplateCommands: %v", err)
	}
	suffixes := map[string]int{".sh": 1, ".md": 1, ".go": 1}
	selected := selectInitCommands(template, suffixes)
	var names []string
	for _, command := range selected {
		names = append(names, command[0])
	}
	want := []string{"prettier", "shfmt", "gofmt", "goimports", "markdownlint"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selected = %v, want %v", names, want)
	}

	// shfmt keeps its template arguments verbatim.
	for _, command := range selected {
		if command[0] == "shfmt" {
			if !reflect.DeepEqual(command, []string{"shfmt", "-i", "2"}) {
				t.Errorf("shfmt command = %v, want [shfmt -i 2]", command)
			}
		}
	}
}

func TestConfirmOverwriteParsing(t *testing.T) {
	for _, test := range []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"YES\n", true},
		{" y \n", true},
		{"n\n", false},
		{"no\n", false},
		{"\n", false},
		{"maybe\n", false},
		{"", false},
	} {
		if got := confirmOverwrite(strings.NewReader(test.input), io.Discard, "x"); got != test.want {
			t.Errorf("confirmOverwrite(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestInitCLIDispatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.sh"), "echo hi\n")
	t.Chdir(root)

	code, stdout, _ := runCLI(t, "init")
	if code != ExitOK {
		t.Fatalf("init exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "Examining current directory... found 1 files across 1 file extensions.") {
		t.Errorf("stdout = %q, want the Examining line", stdout)
	}
	if !strings.Contains(stdout, "Wrote .formattington.json with 1 command(s):") {
		t.Errorf("stdout = %q, want the Wrote line", stdout)
	}
	if !strings.Contains(stdout, "  shfmt -i 2\n") {
		t.Errorf("stdout = %q, want shfmt command", stdout)
	}

	// The written file must be a valid config declaring only shfmt.
	var parsed map[string]any
	data, err := os.ReadFile(filepath.Join(root, ".formattington.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("written config is not valid JSON: %v", err)
	}
	commands, err := validateConfig(parsed, ".formattington.json")
	if err != nil {
		t.Fatalf("written config does not validate: %v", err)
	}
	if !reflect.DeepEqual(commands, [][]string{{"shfmt", "-i", "2"}}) {
		t.Errorf("commands = %v, want [[shfmt -i 2]]", commands)
	}
}

func TestInitCLIRejectsBadInvocations(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"init", "extra"},
		{"--check", "init"},
		{"init", "--check"},
		{"--dry-run", "init"},
		{"--series", "init"},
		{"--config", "some.json", "init"},
		{"--config=some.json", "init"},
	} {
		code, _, _ := runCLI(t, args...)
		if code != ExitUsage {
			t.Errorf("%v: exit = %d, want %d (ExitUsage)", args, code, ExitUsage)
		}
	}
}

func TestInitCLIRefusesToOverwriteWithoutTerminal(t *testing.T) {
	root := t.TempDir()
	sentinel := `{"commands": []}` + "\n"
	writeFile(t, filepath.Join(root, ".formattington.json"), sentinel)
	t.Chdir(root)

	// A pipe is never a terminal, on any platform.
	pipeRead, pipeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pipeRead.Close()
	defer pipeWrite.Close()
	previous := os.Stdin
	os.Stdin = pipeRead
	defer func() { os.Stdin = previous }()

	code, _, _ := runCLI(t, "init")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d (ExitUsage)", code, ExitUsage)
	}
	data, err := os.ReadFile(filepath.Join(root, ".formattington.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != sentinel {
		t.Errorf("existing config was modified: %q, want %q", data, sentinel)
	}
}

// TestRepoConfigIsValid checks the repository's own .formattington.json
// (generated by 'formattington init'): it parses, validates, and declares
// each registered tool at most once.
func TestRepoConfigIsValid(t *testing.T) {
	path := filepath.Join("..", "..", ".formattington.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	commands, err := validateConfig(parsed, path)
	if err != nil {
		t.Fatalf("validate %s: %v", path, err)
	}
	if len(commands) == 0 {
		t.Fatal("repository .formattington.json has no commands")
	}

	seen := map[string]int{}
	for _, command := range commands {
		seen[command[0]]++
	}
	registered := registeredToolNames()
	for name, count := range seen {
		if !registered[name] {
			t.Errorf("%s declares %q, which is not a registered tool", path, name)
		}
		if count > 1 {
			t.Errorf("%s declares %q %d times, want at most once", path, name, count)
		}
	}
}

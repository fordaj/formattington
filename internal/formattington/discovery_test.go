package formattington

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mixedFixture creates a mixed-language fixture with ignored directories.
func mixedFixture(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"main.c":                  "int main(void);\n",
		"util.cpp":                "int util() { return 1; }\n",
		"run.sh":                  "#!/bin/sh\necho hi\n",
		"step.bash":               "#!/bin/bash\necho hi\n",
		"notes.md":                "# notes\n",
		"data.txt":                "hello\n",
		"sub dir/my file.sh":      "#!/bin/sh\necho nested\n",
		"sub dir/my file.cpp":     "int x;\n",
		".git/ignored.sh":         "#!/bin/sh\necho ignored\n",
		"__pycache__/ignored.sh":  "#!/bin/sh\necho ignored\n",
		"node_modules/ignored.sh": "#!/bin/sh\necho ignored\n",
		"tools/ignored.sh":        "#!/bin/sh\necho ignored\n",
		".hidden/ignored.sh":      "#!/bin/sh\necho ignored\n",
	}
	for name, content := range files {
		writeFile(t, filepath.Join(root, name), content)
	}
}

func TestDiscoverFilesFindsOnlyMatchingExtensions(t *testing.T) {
	root := t.TempDir()
	mixedFixture(t, root)

	got := DiscoverFiles([]string{".sh", ".bash"}, root)
	want := []string{"run.sh", "step.bash", filepath.Join("sub dir", "my file.sh")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverFiles = %v, want %v", got, want)
	}
}

func TestDiscoverFilesFindsCAndCppExtensions(t *testing.T) {
	root := t.TempDir()
	mixedFixture(t, root)

	got := DiscoverFiles([]string{".c", ".cpp"}, root)
	want := []string{"main.c", filepath.Join("sub dir", "my file.cpp"), "util.cpp"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverFiles = %v, want %v", got, want)
	}
}

func TestDiscoverFilesSkipsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	mixedFixture(t, root)

	got := DiscoverFiles([]string{".sh"}, root)
	want := []string{"run.sh", filepath.Join("sub dir", "my file.sh")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverFiles = %v, want %v", got, want)
	}
}

func TestDiscoverFilesPreservesSpacesAsSingleEntries(t *testing.T) {
	root := t.TempDir()
	mixedFixture(t, root)

	got := DiscoverFiles([]string{".sh"}, root)
	found := false
	for _, path := range got {
		if path == filepath.Join("sub dir", "my file.sh") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the spaced path in %v", got)
	}
}

func TestDiscoverFilesEmptyDirectory(t *testing.T) {
	if got := DiscoverFiles([]string{".sh"}, t.TempDir()); len(got) != 0 {
		t.Errorf("DiscoverFiles = %v, want no results", got)
	}
}

func TestDiscoverFilesMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	if got := DiscoverFiles([]string{".sh"}, root); len(got) != 0 {
		t.Errorf("DiscoverFiles = %v, want no results", got)
	}
}

func TestDiscoverFilesIsCaseInsensitive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "UPPER.SH"), "#!/bin/sh\n")

	got := DiscoverFiles([]string{".sh"}, root)
	if !reflect.DeepEqual(got, []string{"UPPER.SH"}) {
		t.Errorf("DiscoverFiles = %v, want [UPPER.SH]", got)
	}
}

func TestDiscoverFilesResultsAreSorted(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b.sh", "a.sh", "c.sh"} {
		writeFile(t, filepath.Join(root, name), "#!/bin/sh\n")
	}

	got := DiscoverFiles([]string{".sh"}, root)
	if !reflect.DeepEqual(got, []string{"a.sh", "b.sh", "c.sh"}) {
		t.Errorf("DiscoverFiles = %v, want [a.sh b.sh c.sh]", got)
	}
}

func TestDiscoverFilesUsesWorkingDirectoryByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "run.sh"), "#!/bin/sh\n")
	t.Chdir(root)

	got := DiscoverFiles([]string{".sh"}, "")
	if !reflect.DeepEqual(got, []string{"run.sh"}) {
		t.Errorf("DiscoverFiles = %v, want [run.sh]", got)
	}
}

func TestRegistryCoversEveryExtension(t *testing.T) {
	for _, tool := range registry {
		if len(tool.Extensions) == 0 {
			t.Errorf("tool %q has no extensions", tool.Name)
		}
	}
}

func TestExtensionOwnership(t *testing.T) {
	cases := map[string][]string{
		"shfmt":        {".sh", ".bash"},
		"black":        {".py"},
		"taplo":        {".toml"},
		"gofmt":        {".go"},
		"goimports":    {".go"},
		"powershell":   {".ps1", ".psm1"},
		"clang-format": {".c", ".cpp", ".hpp"},
	}
	for name, wanted := range cases {
		tool := lookupTool(name)
		for _, ext := range wanted {
			if !contains(tool.Extensions, ext) {
				t.Errorf("%q does not handle %q", name, ext)
			}
		}
	}
}

func TestFormatterExtensionsAreDistinct(t *testing.T) {
	if len(prettierExtensions) != 12 {
		t.Errorf("prettier handles %d extensions, want 12", len(prettierExtensions))
	}
	if !contains(prettierExtensions, ".md") || !contains(prettierExtensions, ".markdown") {
		t.Error("prettier must own Markdown formatting")
	}
}

func TestIgnoredDirectoryNames(t *testing.T) {
	for _, name := range []string{
		"__pycache__", "node_modules", "vendor", "venv", "dist", "build", "tools",
		"formatters",
	} {
		if !ignoredDirNames[name] {
			t.Errorf("%q must be ignored by discovery", name)
		}
	}
}

func TestIsIgnoredDirName(t *testing.T) {
	if !isIgnoredDirName(".hidden") {
		t.Error("hidden directories must be ignored")
	}
	if !isIgnoredDirName("node_modules") {
		t.Error("node_modules must be ignored")
	}
	if isIgnoredDirName("src") {
		t.Error("regular directories must not be ignored")
	}
}

func TestMainWithNoArgumentsIsANoop(t *testing.T) {
	fakeTools(t)
	t.Chdir(t.TempDir())

	code, stdout, stderr := runCLI(t)
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("expected no output, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestMainWithoutFormattersDoesNotRunTools(t *testing.T) {
	_, argvFile, _ := fakeTools(t, "shfmt", "black")
	t.Chdir(t.TempDir())

	code, _, _ := runCLI(t, "--series")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if got := recordedArgv(t, argvFile); len(got) != 0 {
		t.Errorf("expected no tool invocations, got %v", got)
	}
}

func TestMainHonorsWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.sh"), "#!/bin/sh\n")
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if got := DiscoverFiles([]string{".sh"}, ""); len(got) != 1 {
		t.Errorf("expected one discovered file, got %v", got)
	}
}

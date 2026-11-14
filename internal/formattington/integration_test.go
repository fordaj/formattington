package formattington

import (
	"path/filepath"
	"reflect"
	"testing"
)

// installBundled creates a bundled fake tool and points the resolver at it.
func installBundled(t *testing.T, name, version, platform, exe string) (toolsDir, argvFile string) {
	t.Helper()
	root := t.TempDir()
	toolsDir = filepath.Join(root, "tools")
	bundled := filepath.Join(toolsDir, name, "v"+version, platform, exe)
	writeFile(t, bundled, fakeExecutableScript)
	if err := chmodExecutable(bundled); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	argvFile = filepath.Join(root, "argv.txt")
	writeFile(t, argvFile, "")

	t.Setenv("FORMATTINGTON_TOOLS_DIR", toolsDir)
	t.Setenv("FORMATTINGTON_VERBOSE", "")
	t.Setenv("FAKE_ARGV_FILE", argvFile)
	t.Setenv("FAKE_CALLED_FILE", filepath.Join(root, "called"))
	t.Setenv("PATH", filepath.Join(root, "bin"))
	return toolsDir, argvFile
}

func TestBundledFakeIsResolved(t *testing.T) {
	requirePOSIX(t)

	_, argvFile := installBundled(t, "shfmt", "3.14.1", "macos-x86_64", "shfmt")

	code, err := runShfmt(Options{})
	if err != nil {
		t.Fatalf("runShfmt: %v", err)
	}
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if got := recordedArgv(t, argvFile); !reflect.DeepEqual(got, []string{"-w"}) {
		t.Errorf("argv = %v, want [-w]", got)
	}
}

func TestBundledFakeBatchesManyPathsInOneInvocation(t *testing.T) {
	requirePOSIX(t)

	_, argvFile := installBundled(t, "shfmt", "3.14.1", "macos-x86_64", "shfmt")

	paths := []string{"a.sh", "b with spaces.sh", "very/long/path/to/c.sh", "d.sh"}
	if _, err := runShfmt(Options{Args: paths}); err != nil {
		t.Fatalf("runShfmt: %v", err)
	}

	recorded := recordedArgv(t, argvFile)
	want := append([]string{"-w"}, paths...)
	if !reflect.DeepEqual(recorded, want) {
		t.Errorf("argv = %v, want %v", recorded, want)
	}
}

func TestBundledFakeCheckMode(t *testing.T) {
	requirePOSIX(t)

	_, argvFile := installBundled(t, "shfmt", "3.14.1", "macos-x86_64", "shfmt")

	if _, err := runShfmt(Options{Check: true}); err != nil {
		t.Fatalf("runShfmt: %v", err)
	}
	if got := recordedArgv(t, argvFile); !reflect.DeepEqual(got, []string{"-l"}) {
		t.Errorf("argv = %v, want [-l]", got)
	}
}

func TestPathStubIsUsedAsFallback(t *testing.T) {
	requirePOSIX(t)

	_, argvFile, _ := fakeTools(t, "shfmt")

	if _, err := runShfmt(Options{Args: []string{"file with spaces.sh"}}); err != nil {
		t.Fatalf("runShfmt: %v", err)
	}
	got := recordedArgv(t, argvFile)
	want := []string{"-w", "file with spaces.sh"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("argv = %v, want %v", got, want)
	}
}

func TestCLIEndToEnd(t *testing.T) {
	requirePOSIX(t)

	cases := []struct {
		name  string
		tools []string
		args  []string
		want  []string
	}{
		{"shfmt check", []string{"shfmt"}, []string{"--check", "shfmt"}, []string{"-l"}},
		{"clang-format check", []string{"clang-format"}, []string{"--check", "clang-format"}, []string{"--dry-run", "--Werror"}},
		{"black check", []string{"black"}, []string{"--check", "black", "my file.py"}, []string{"--check", "my file.py"}},
		{"prettier check", []string{"prettier"}, []string{"--check", "prettier", "my file.js"}, []string{"--check", "my file.js"}},
		{"prettier write", []string{"prettier"}, []string{"prettier"}, []string{"--write"}},
		{"gofmt check", []string{"gofmt"}, []string{"--check", "gofmt", "my file.go"}, []string{"-l", "my file.go"}},
		{"goimports dry run", []string{"goimports"}, []string{"--dry-run", "goimports", "file.go"}, []string{"-d", "file.go"}},
		{"taplo check", []string{"taplo"}, []string{"--check", "toml"}, []string{"fmt", "--colors", "never", "--check"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, argvFile, _ := fakeTools(t, testCase.tools...)
			t.Chdir(t.TempDir())

			code, _, stderr := runCLI(t, testCase.args...)
			if code != ExitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
			}
			got := recordedArgv(t, argvFile)
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("argv = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestCLIDiscoveryEndToEnd(t *testing.T) {
	requirePOSIX(t)

	root := t.TempDir()
	mixedFixture(t, root)
	_, argvFile, _ := fakeTools(t, "shfmt", "clang-format")
	t.Chdir(root)

	code, _, stderr := runCLI(t, "shfmt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	got := recordedArgv(t, argvFile)
	want := []string{"-w", "run.sh", "step.bash", filepath.Join("sub dir", "my file.sh")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("argv = %v, want %v", got, want)
	}
}

func TestCLIExplicitFilesSuppressDiscovery(t *testing.T) {
	requirePOSIX(t)

	root := t.TempDir()
	mixedFixture(t, root)
	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(root)

	code, _, _ := runCLI(t, "shfmt", "run.sh")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	got := recordedArgv(t, argvFile)
	want := []string{"-w", "run.sh"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("argv = %v, want %v", got, want)
	}
}

func TestCLIDiscoveryWithNoMatchesStillRunsOnce(t *testing.T) {
	requirePOSIX(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "notes.md"), "# notes\n")
	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(root)

	code, _, _ := runCLI(t, "shfmt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	got := recordedArgv(t, argvFile)
	if !reflect.DeepEqual(got, []string{"-w"}) {
		t.Errorf("argv = %v, want [-w]", got)
	}
}

func TestBatchesRespectTheMaximumFileCount(t *testing.T) {
	requirePOSIX(t)

	root := t.TempDir()
	for index := 0; index < 5; index++ {
		writeFile(t, filepath.Join(root, "f"+string(rune('0'+index))+".sh"), "#!/bin/sh\n")
	}
	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(root)

	// The batch limit is a package constant; a smaller one keeps the
	// test fast while exercising the same code path.
	original := maxFilesPerInvocation
	t.Cleanup(func() { maxFilesPerInvocation = original })
	maxFilesPerInvocation = 2

	code, _, _ := runCLI(t, "shfmt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}

	recorded := recordedArgv(t, argvFile)
	want := []string{"-w", "f0.sh", "f1.sh", "-w", "f2.sh", "f3.sh", "-w", "f4.sh"}
	if !reflect.DeepEqual(recorded, want) {
		t.Errorf("argv = %v, want %v", recorded, want)
	}
}

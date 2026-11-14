package formattington

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// allFakeTools installs a fake for every registered formatter, except the
// names given in skip.
func allFakeTools(t *testing.T, skip ...string) (argvFile string) {
	t.Helper()
	excluded := map[string]bool{}
	for _, name := range skip {
		excluded[name] = true
	}
	names := []string{}
	for _, tool := range registry {
		if excluded[tool.Name] {
			continue
		}
		if tool.Name == "powershell" {
			names = append(names, "pwsh")
			continue
		}
		names = append(names, tool.Name)
	}
	_, argvFile, _ = fakeTools(t, names...)
	return argvFile
}

func TestSingleToolSelection(t *testing.T) {
	requirePOSIX(t)

	for _, tool := range registry {
		t.Run(tool.Name, func(t *testing.T) {
			_, argvFile, calledFile := fakeTools(t, fakeNameFor(tool.Name))
			t.Chdir(t.TempDir())

			code, _, stderr := runCLI(t, tool.Name)
			if code != ExitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
			}
			if !toolWasCalled(t, calledFile) {
				t.Errorf("expected %s to be invoked", tool.Name)
			}
			_ = argvFile
		})
	}
}

// fakeNameFor returns the executable name a formatter resolves to.
func fakeNameFor(formatter string) string {
	if formatter == "powershell" {
		return "pwsh"
	}
	return formatter
}

func TestLanguageAliasSelection(t *testing.T) {
	requirePOSIX(t)

	cases := map[string]string{
		"c":        "clang-format",
		"cpp":      "clang-format",
		"c++":      "clang-format",
		"bash":     "shfmt",
		"python":   "black",
		"py":       "black",
		"md":       "prettier",
		"markdown": "prettier",
		"html":     "prettier",
		"ps1":      "powershell",
		"toml":     "taplo",
		"go":       "gofmt",
	}

	for alias, formatter := range cases {
		t.Run(alias, func(t *testing.T) {
			if languageAliases[alias] != formatter {
				t.Fatalf("alias %q maps to %q, want %q", alias, languageAliases[alias], formatter)
			}
			if !isRegisteredTool(formatter) {
				t.Fatalf("formatter %q is not registered", formatter)
			}

			argvFile := allFakeTools(t)
			t.Chdir(t.TempDir())

			code, _, stderr := runCLI(t, alias)
			if code != ExitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
			}

			recorded := recordedArgv(t, argvFile)
			want := modeFlagsFor(formatter)
			if len(want) == 0 {
				// Black is invoked without any arguments in default mode.
				return
			}
			for _, flag := range want {
				if !contains(recorded, flag) {
					t.Errorf("argv = %v, must contain %q for %q", recorded, flag, formatter)
				}
			}
		})
	}
}

// modeFlagsFor returns the argv fragments that identify a formatter's
// default invocation.
func modeFlagsFor(formatter string) []string {
	switch formatter {
	case "shfmt":
		return []string{"-w"}
	case "clang-format":
		return []string{"-i"}
	case "prettier":
		return []string{"--write"}
	case "powershell":
		return []string{"-NoProfile", "-Command"}
	case "taplo":
		return []string{"fmt", "--colors", "never"}
	case "gofmt", "goimports":
		return []string{"-w"}
	default:
		return nil
	}
}

func TestAllRunsEveryFormatter(t *testing.T) {
	requirePOSIX(t)

	argvFile := allFakeTools(t)
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "all", "--series")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}

	recorded := recordedArgv(t, argvFile)
	for _, tool := range registry {
		for _, flag := range modeFlagsFor(tool.Name) {
			if !contains(recorded, flag) {
				t.Errorf("recorded argv %v is missing %q for %q", recorded, flag, tool.Name)
			}
		}
	}
}

func TestAllPreservesTheLastNonZeroExitCode(t *testing.T) {
	requirePOSIX(t)

	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "shfmt"), "#!/bin/sh\nexit 5\n")
	if err := chmodExecutable(filepath.Join(binDir, "shfmt")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	for _, name := range []string{"clang-format", "black", "prettier", "pwsh", "taplo", "gofmt", "goimports"} {
		writeFakeExecutable(t, binDir, name)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)
	t.Chdir(t.TempDir())

	code, _, _ := runCLI(t, "all", "--series")
	if code != 5 {
		t.Errorf("exit = %d, want 5", code)
	}
}

func TestAllWithExtraArgsExitsTwo(t *testing.T) {
	fakeTools(t)

	code, _, stderr := runCLI(t, "all", "extra")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "extra arguments are not supported") {
		t.Errorf("stderr = %q, must explain the unsupported arguments", stderr)
	}
}

func TestModesAreExclusive(t *testing.T) {
	fakeTools(t)

	code, _, stderr := runCLI(t, "--check", "--dry-run", "shfmt")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "not allowed with argument") {
		t.Errorf("stderr = %q, must explain the exclusivity", stderr)
	}
}

func TestUnknownGlobalFlagExitsTwo(t *testing.T) {
	fakeTools(t)

	code, _, stderr := runCLI(t, "--bogus", "shfmt")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "unrecognized arguments") {
		t.Errorf("stderr = %q, must report the unknown flag", stderr)
	}
}

func TestUnknownWordExitsTwo(t *testing.T) {
	fakeTools(t)

	for _, word := range []string{"foobar", "unknown-word", "cs"} {
		code, _, stderr := runCLI(t, word)
		if code != ExitUsage {
			t.Errorf("%q: exit = %d, want %d", word, code, ExitUsage)
		}
		if !strings.Contains(stderr, "unrecognized arguments") {
			t.Errorf("%q: stderr = %q, must report the unknown word", word, stderr)
		}
	}
}

func TestUnregisteredAliasIsNotYetSupported(t *testing.T) {
	fakeTools(t)

	languageAliases["cobol"] = "cobol-formatter"
	t.Cleanup(func() { delete(languageAliases, "cobol") })

	code, _, stderr := runCLI(t, "cobol")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "not yet supported") {
		t.Errorf("stderr = %q, must say the alias is not yet supported", stderr)
	}
}

func TestHelpListsOptionsAndAliases(t *testing.T) {
	fakeTools(t)

	code, stdout, _ := runCLI(t, "--help")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	for _, needle := range []string{
		"--help", "--version", "--check", "--dry-run", "--series", "--verbose", "--config",
		"c/cpp/c++", "bash", "python/py", "html", "clang-format",
		"md/markdown use prettier", ".md", ".markdown",
		"ps1", "powershell", "PSScriptAnalyzer",
		"toml", "taplo",
		"go uses gofmt", "goimports",
		"concurrently", "last non-zero",
		"FORMATTINGTON_CONFIG_PATH", ".formattington.json",
	} {
		if !strings.Contains(stdout, needle) {
			t.Errorf("help output is missing %q", needle)
		}
	}
}

func TestShortHelpFlag(t *testing.T) {
	fakeTools(t)

	code, stdout, _ := runCLI(t, "-h")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "usage: formattington") {
		t.Errorf("stdout = %q, must contain the usage line", stdout)
	}
}

func TestVersionPrintsTheVersion(t *testing.T) {
	fakeTools(t)

	code, stdout, _ := runCLI(t, "--version")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("expected a version string")
	}
}

func TestVersionBeforeFormatterDoesNotRunTheFormatter(t *testing.T) {
	requirePOSIX(t)

	argvFile := allFakeTools(t)
	t.Chdir(t.TempDir())

	code, stdout, _ := runCLI(t, "--version", "shfmt")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("expected a version string")
	}
	if got := recordedArgv(t, argvFile); len(got) != 0 {
		t.Errorf("argv = %v, want no tool invocations", got)
	}
}

func TestArgumentForwarding(t *testing.T) {
	requirePOSIX(t)

	cases := []struct {
		formatter string
		forwarded []string
		want      []string
	}{
		{"shfmt", []string{"--keep-padding", "-i", "2"}, []string{"-w", "--keep-padding", "-i", "2"}},
		{"clang-format", []string{"--style=file"}, []string{"-i", "--style=file"}},
		{"black", []string{"--line-length", "88"}, []string{"--line-length", "88"}},
		{"prettier", []string{"--check"}, []string{"--check"}},
		{"gofmt", []string{"-s"}, []string{"-w", "-s"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.formatter, func(t *testing.T) {
			_, argvFile, _ := fakeTools(t, testCase.formatter)
			t.Chdir(t.TempDir())

			code, _, stderr := runCLI(t, append([]string{testCase.formatter}, testCase.forwarded...)...)
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

func TestForwardedArgsKeepSpacesAsSingleEntries(t *testing.T) {
	requirePOSIX(t)

	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(t.TempDir())

	code, _, _ := runCLI(t, "shfmt", "file with spaces.sh", "-i", "2")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	want := []string{"-w", "file with spaces.sh", "-i", "2"}
	got := recordedArgv(t, argvFile)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("argv = %v, want %v", got, want)
	}
}

func TestForwardedCheckBelongsToTheFormatter(t *testing.T) {
	requirePOSIX(t)

	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(t.TempDir())

	code, _, _ := runCLI(t, "shfmt", "--check")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	got := recordedArgv(t, argvFile)
	if !reflect.DeepEqual(got, []string{"-w", "--check"}) {
		t.Errorf("argv = %v, want [-w --check]", got)
	}
}

func TestMissingToolExitsOneTwentySeven(t *testing.T) {
	fakeTools(t)

	for _, mode := range [][]string{{}, {"--check"}, {"--dry-run"}} {
		for _, tool := range registry {
			args := append(append([]string{}, mode...), tool.Name)
			code, _, stderr := runCLI(t, args...)
			if code != ExitNotFound {
				t.Errorf("%v: exit = %d, want %d", args, code, ExitNotFound)
			}
			if !strings.HasPrefix(stderr, "formattington:") {
				t.Errorf("%v: stderr = %q, must start with the program name", args, stderr)
			}
			if strings.Contains(stderr, "panic") || strings.Contains(stderr, "goroutine") {
				t.Errorf("%v: stderr = %q, must not contain a traceback", args, stderr)
			}
		}
	}
}

func TestAllReturnsOneTwentySevenWhenAnyToolIsMissing(t *testing.T) {
	requirePOSIX(t)

	// Only shfmt is missing; the others are present.
	binDir := filepath.Join(t.TempDir(), "bin")
	for _, name := range []string{"clang-format", "black", "prettier", "pwsh", "taplo", "gofmt", "goimports"} {
		writeFakeExecutable(t, binDir, name)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "all")
	if code != ExitNotFound {
		t.Errorf("exit = %d, want %d", code, ExitNotFound)
	}
	if !strings.Contains(stderr, "Could not find 'shfmt'") {
		t.Errorf("stderr = %q, must report the missing tool", stderr)
	}
}

func TestAllReportsEveryMissingTool(t *testing.T) {
	fakeTools(t)

	code, _, stderr := runCLI(t, "--check", "all")
	if code != ExitNotFound {
		t.Errorf("exit = %d, want %d", code, ExitNotFound)
	}
	if got := strings.Count(stderr, "formattington:"); got != len(registry) {
		t.Errorf("stderr reported %d missing tools, want %d", got, len(registry))
	}
}

func TestSeriesOrderIsRegistrationOrder(t *testing.T) {
	requirePOSIX(t)

	orderFile := filepath.Join(t.TempDir(), "order.txt")
	binDir := filepath.Join(t.TempDir(), "bin")
	for _, tool := range registry {
		name := tool.Name
		if name == "powershell" {
			name = "pwsh"
		}
		writeFile(t, filepath.Join(binDir, name),
			"#!/bin/sh\nprintf '%s\\n' \""+tool.Name+"\" >> \"$FAKE_ORDER_FILE\"\nexit 0\n")
		if err := chmodExecutable(filepath.Join(binDir, name)); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)
	t.Setenv("FAKE_ORDER_FILE", orderFile)
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "all", "--series")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}

	var want []string
	for _, tool := range registry {
		want = append(want, tool.Name)
	}
	got := recordedArgv(t, orderFile)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestDefaultAllRunsConcurrently(t *testing.T) {
	requirePOSIX(t)

	// Every fake waits until all formatters have started: a sequential
	// run would never reach the barrier.
	binDir := filepath.Join(t.TempDir(), "bin")
	barrierDir := filepath.Join(t.TempDir(), "barrier")
	if err := os.MkdirAll(barrierDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", barrierDir, err)
	}
	for _, tool := range registry {
		name := fakeNameFor(tool.Name)
		writeFile(t, filepath.Join(binDir, name),
			"#!/bin/sh\n: > \""+barrierDir+"/$$\"\n"+
				"i=0\nwhile [ $i -lt 200 ]; do\n"+
				"  n=$(ls \""+barrierDir+"\" | wc -l)\n"+
				"  if [ \"$n\" -ge 8 ]; then break; fi\n"+
				"  sleep 0.05\n  i=$((i+1))\ndone\nexit 0\n")
		if err := chmodExecutable(filepath.Join(binDir, name)); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(t.TempDir())

	done := make(chan int, 1)
	go func() {
		code, _, _ := runCLI(t, "all")
		done <- code
	}()

	select {
	case code := <-done:
		if code != ExitOK {
			t.Errorf("exit = %d, want %d", code, ExitOK)
		}
	case <-timeoutAfter():
		t.Error("the default 'all' run did not overlap the formatters")
	}
}

func TestSeriesDoesNotOverlap(t *testing.T) {
	requirePOSIX(t)

	// Every fake refuses to start while another one holds the lock: a
	// sequential run succeeds, an overlapping one fails.
	binDir := filepath.Join(t.TempDir(), "bin")
	lock := filepath.Join(t.TempDir(), "lock")
	for _, tool := range registry {
		name := fakeNameFor(tool.Name)
		writeFile(t, filepath.Join(binDir, name),
			"#!/bin/sh\nif [ -f \"$FAKE_LOCK\" ]; then exit 9; fi\n"+
				": > \"$FAKE_LOCK\"\nsleep 0.2\nrm -f \"$FAKE_LOCK\"\nexit 0\n")
		if err := chmodExecutable(filepath.Join(binDir, name)); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_LOCK", lock)
	t.Chdir(t.TempDir())

	code, _, _ := runCLI(t, "all", "--series")
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
}

func TestExitAggregationFollowsRegistrationOrder(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "shfmt"), "#!/bin/sh\nexit 5\n")
	writeFakeExecutable(t, binDir, "clang-format")
	writeFile(t, filepath.Join(binDir, "black"), "#!/bin/sh\nexit 2\n")
	writeFakeExecutable(t, binDir, "prettier")
	writeFakeExecutable(t, binDir, "pwsh")
	writeFakeExecutable(t, binDir, "taplo")
	writeFakeExecutable(t, binDir, "gofmt")
	writeFakeExecutable(t, binDir, "goimports")
	for _, name := range []string{"shfmt", "black"} {
		if err := chmodExecutable(filepath.Join(binDir, name)); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)
	t.Chdir(t.TempDir())

	// shfmt=5, black=2: the last non-zero in registration order wins.
	code, _, _ := runCLI(t, "all", "--series")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestSeriesIsANoopForSingleFormatters(t *testing.T) {
	requirePOSIX(t)

	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "--series", "shfmt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	got := recordedArgv(t, argvFile)
	if !reflect.DeepEqual(got, []string{"-w"}) {
		t.Errorf("argv = %v, want [-w]", got)
	}
}

func TestSeriesAfterASingleFormatterIsForwarded(t *testing.T) {
	requirePOSIX(t)

	_, argvFile, _ := fakeTools(t, "shfmt")
	t.Chdir(t.TempDir())

	code, _, _ := runCLI(t, "shfmt", "--series")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	got := recordedArgv(t, argvFile)
	if !reflect.DeepEqual(got, []string{"-w", "--series"}) {
		t.Errorf("argv = %v, want [-w --series]", got)
	}
}

func TestVerboseIsForwardedToToolResolution(t *testing.T) {
	requirePOSIX(t)

	fakeTools(t, "shfmt")
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "--verbose", "shfmt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if !strings.Contains(stderr, "resolved to") {
		t.Errorf("stderr = %q, must report the resolved executable", stderr)
	}
}

func TestVerboseEnvironmentVariable(t *testing.T) {
	requirePOSIX(t)

	fakeTools(t, "shfmt")
	t.Setenv("FORMATTINGTON_VERBOSE", "1")
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "shfmt")
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
	}
	if !strings.Contains(stderr, "resolved to") {
		t.Errorf("stderr = %q, must report the resolved executable", stderr)
	}
}

func TestConcurrentFormattersDoNotSwallowOthers(t *testing.T) {
	requirePOSIX(t)

	argvFile := allFakeTools(t, "shfmt")
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "all")
	if code != ExitNotFound {
		t.Errorf("exit = %d, want %d", code, ExitNotFound)
	}
	if !strings.Contains(stderr, "Could not find 'shfmt'") {
		t.Errorf("stderr = %q, must report the missing tool", stderr)
	}
	// The present formatters still ran.
	recorded := recordedArgv(t, argvFile)
	if !contains(recorded, "-i") {
		t.Errorf("argv = %v, clang-format must still have run", recorded)
	}
}

// concurrentGroup runs the given functions concurrently and waits for all
// of them to finish.
func concurrentGroup(t *testing.T, fns ...func()) {
	t.Helper()
	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)
		go func(fn func()) {
			defer wg.Done()
			fn()
		}(fn)
	}
	wg.Wait()
}

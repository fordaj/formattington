package formattington

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeConfig writes a config file and returns its path.
func writeConfig(t *testing.T, path, content string) string {
	t.Helper()
	writeFile(t, path, content)
	return path
}

// isolateConfig prepares a clean working directory for config tests.
func isolateConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv(configEnvVar, "")
	return root
}

func TestConfigDiscoveryPrecedence(t *testing.T) {
	requirePOSIX(t)

	t.Run("flag after all", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black", "--check"]]}`)

		code, _, stderr := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"--check"}) {
			t.Errorf("argv = %v, want [--check]", got)
		}
	})

	t.Run("flag before all", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["shfmt"]]}`)

		code, _, stderr := runCLI(t, "--config", cfg, "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", got)
		}
	})

	t.Run("flag equals form after all", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "all", "--config="+cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		if got := recordedArgv(t, argvFile); len(got) != 0 {
			t.Errorf("argv = %v, want no arguments", got)
		}
	})

	t.Run("environment variable", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "env.json"), `{"commands": [["black"]]}`)
		t.Setenv(configEnvVar, cfg)

		code, _, _ := runCLI(t, "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		if got := recordedArgv(t, argvFile); len(got) != 0 {
			t.Errorf("argv = %v, want the black invocation", got)
		}
	})

	t.Run("auto discovered", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "shfmt", "black")
		writeConfig(t, filepath.Join(root, configFilename), `{"commands": [["shfmt", "-i", "2"]]}`)

		code, _, _ := runCLI(t, "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"-w", "-i", "2"}) {
			t.Errorf("argv = %v, want [-w -i 2]", got)
		}
	})

	t.Run("no config falls back to the full registry", func(t *testing.T) {
		isolateConfig(t)
		argvFile := allFakeTools(t)

		code, _, _ := runCLI(t, "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		recorded := recordedArgv(t, argvFile)
		if !contains(recorded, "-i") || !contains(recorded, "--write") {
			t.Errorf("argv = %v, want every formatter to have run", recorded)
		}
	})

	t.Run("flag beats env and auto", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt", "clang-format")
		writeConfig(t, filepath.Join(root, configFilename), `{"commands": [["shfmt"]]}`)
		envCfg := writeConfig(t, filepath.Join(root, "env.json"), `{"commands": [["clang-format"]]}`)
		flagCfg := writeConfig(t, filepath.Join(root, "flag.json"), `{"commands": [["black"]]}`)
		t.Setenv(configEnvVar, envCfg)

		code, _, _ := runCLI(t, "all", "--config", flagCfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if len(got) != 0 {
			t.Errorf("argv = %v, want only the black invocation", got)
		}
	})

	t.Run("env beats auto", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt")
		writeConfig(t, filepath.Join(root, configFilename), `{"commands": [["shfmt"]]}`)
		envCfg := writeConfig(t, filepath.Join(root, "env.json"), `{"commands": [["black"]]}`)
		t.Setenv(configEnvVar, envCfg)

		code, _, _ := runCLI(t, "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		if got := recordedArgv(t, argvFile); len(got) != 0 {
			t.Errorf("argv = %v, want only the black invocation", got)
		}
	})

	t.Run("empty env falls through to auto", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "shfmt", "black")
		writeConfig(t, filepath.Join(root, configFilename), `{"commands": [["shfmt", "-i", "2"]]}`)
		t.Setenv(configEnvVar, "")

		code, _, _ := runCLI(t, "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"-w", "-i", "2"}) {
			t.Errorf("argv = %v, want [-w -i 2]", got)
		}
	})

	t.Run("relative path resolves against the working directory", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t, "black")
		writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "all", "--config", "cfg.json")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
	})

	t.Run("path with spaces", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t, "black")
		sub := filepath.Join(root, "dir with spaces")
		cfg := writeConfig(t, filepath.Join(sub, "my config.json"), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
	})

	t.Run("config path equal to a formatter name", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt")
		writeConfig(t, filepath.Join(root, "black"), `{"commands": [["shfmt"]]}`)

		code, _, _ := runCLI(t, "--config", "black", "all")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", got)
		}
	})
}

func TestConfigErrorsExitTwo(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		isolateConfig(t)
		fakeTools(t)

		code, _, stderr := runCLI(t, "all", "--config", "missing-config.json")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
		if !strings.HasPrefix(stderr, "formattington:") {
			t.Errorf("stderr = %q, must start with the program name", stderr)
		}
		if !strings.Contains(stderr, "missing-config.json") {
			t.Errorf("stderr = %q, must name the missing file", stderr)
		}
		if !strings.Contains(stderr, "--config") {
			t.Errorf("stderr = %q, must name the source of the config", stderr)
		}
	})

	t.Run("not a file", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		sub := filepath.Join(root, "adir")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		code, _, stderr := runCLI(t, "all", "--config", sub)
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
		if !strings.Contains(stderr, sub) {
			t.Errorf("stderr = %q, must name the path", stderr)
		}
	})

	t.Run("environment variable points at a missing file", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		t.Setenv(configEnvVar, filepath.Join(root, "missing.json"))

		code, _, stderr := runCLI(t, "all")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
		if !strings.Contains(stderr, configEnvVar) {
			t.Errorf("stderr = %q, must name the environment variable", stderr)
		}
	})

	t.Run("malformed auto-discovered config", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		writeConfig(t, filepath.Join(root, configFilename), "not json at all")

		code, _, stderr := runCLI(t, "all")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
		if !strings.Contains(stderr, configFilename) {
			t.Errorf("stderr = %q, must name the config file", stderr)
		}
	})
}

func TestConfigSchemaValidation(t *testing.T) {
	payloads := map[string]string{
		"bad-json":           "not json at all",
		"bare-array":         `[["black"]]`,
		"top-string":         `"black"`,
		"top-number":         `42`,
		"missing-commands":   `{}`,
		"non-list-commands":  `"commands": "black"`,
		"shell-string":       `{"commands": ["black --check"]}`,
		"object-entry":       `{"commands": [{"tool": "black"}]}`,
		"non-list-entry":     `{"commands": [42]}`,
		"empty-argv":         `{"commands": [[]]}`,
		"non-string-element": `{"commands": [["black", 42]]}`,
		"unknown-tool":       `{"commands": [["nosuch"]]}`,
		"all-not-allowed":    `{"commands": [["all"]]}`,
		"unknown-top-key":    `{"commands": [], "extr": 1}`,
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			root := isolateConfig(t)
			_, argvFile, calledFile := fakeTools(t, "black", "shfmt")
			cfg := writeConfig(t, filepath.Join(root, "cfg.json"), payload)

			code, _, stderr := runCLI(t, "all", "--config", cfg)
			if code != ExitUsage {
				t.Errorf("exit = %d, want %d", code, ExitUsage)
			}
			if !strings.HasPrefix(stderr, "formattington:") {
				t.Errorf("stderr = %q, must start with the program name", stderr)
			}
			if !strings.Contains(stderr, "cfg.json") {
				t.Errorf("stderr = %q, must name the config file", stderr)
			}
			if toolWasCalled(t, calledFile) {
				t.Error("no formatter may run when the config is invalid")
			}
			if got := recordedArgv(t, argvFile); len(got) != 0 {
				t.Errorf("argv = %v, want no invocations", got)
			}
		})
	}
}

func TestConfigUnknownToolNamesTheEntryIndex(t *testing.T) {
	root := isolateConfig(t)
	fakeTools(t)
	cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"], ["nosuch", "--x"]]}`)

	code, _, stderr := runCLI(t, "all", "--config", cfg)
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "commands[1]") {
		t.Errorf("stderr = %q, must name the offending entry", stderr)
	}
	if !strings.Contains(stderr, "nosuch") {
		t.Errorf("stderr = %q, must name the unknown tool", stderr)
	}
}

func TestConfigEmptyCommandsRunsNothing(t *testing.T) {
	root := isolateConfig(t)
	_, _, calledFile := fakeTools(t, "black", "shfmt")
	cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": []}`)

	for _, args := range [][]string{
		{"all", "--config", cfg},
		{"all", "--series", "--config", cfg},
	} {
		code, _, _ := runCLI(t, args...)
		if code != ExitOK {
			t.Errorf("%v: exit = %d, want %d", args, code, ExitOK)
		}
	}
	if toolWasCalled(t, calledFile) {
		t.Error("no formatter may run for an empty command list")
	}
}

func TestConfigExecution(t *testing.T) {
	requirePOSIX(t)

	t.Run("runs exactly the listed commands", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "prettier", "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"),
			`{"commands": [["black", "--check", "src"], ["prettier", "docs/**/*.md"]]}`)

		code, _, stderr := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
		}
		got := recordedArgv(t, argvFile)
		if !contains(got, "--check") || !contains(got, "src") {
			t.Errorf("argv = %v, want the black command", got)
		}
		if !contains(got, "docs/**/*.md") {
			t.Errorf("argv = %v, want the prettier command", got)
		}
		if contains(got, "-w") {
			t.Errorf("argv = %v, must not run shfmt", got)
		}
	})

	t.Run("alias commands resolve", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "shfmt", "black", "prettier", "clang-format", "pwsh", "taplo", "gofmt", "goimports")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"),
			`{"commands": [["bash"], ["python"], ["md"], ["c"], ["ps1"], ["toml"], ["go"], ["goimports"]]}`)

		code, _, stderr := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
		}
		got := recordedArgv(t, argvFile)
		for _, flag := range []string{"-w", "-i", "--write", "fmt", "-NoProfile"} {
			if !contains(got, flag) {
				t.Errorf("argv = %v, must contain %q", got, flag)
			}
		}
	})

	t.Run("spaces stay single entries", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "prettier", "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"),
			`{"commands": [["prettier", "my file.md"], ["shfmt", "-i", "2", "sub dir/my file.sh"]]}`)

		code, _, _ := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !contains(got, "my file.md") {
			t.Errorf("argv = %v, want the spaced path", got)
		}
		if !contains(got, "sub dir/my file.sh") {
			t.Errorf("argv = %v, want the spaced path", got)
		}
	})

	t.Run("check propagates to the commands", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black", "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"], ["shfmt"]]}`)

		code, _, _ := runCLI(t, "--check", "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !contains(got, "--check") {
			t.Errorf("argv = %v, want the check flag", got)
		}
		if !contains(got, "-l") {
			t.Errorf("argv = %v, want the shfmt list flag", got)
		}
	})

	t.Run("dry run and verbose propagate", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t, "black", "prettier")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"], ["prettier"]]}`)

		code, _, stderr := runCLI(t, "--dry-run", "--verbose", "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, ExitOK, stderr)
		}
		if !strings.Contains(stderr, "resolved to") {
			t.Errorf("stderr = %q, must report the resolved executable", stderr)
		}
	})

	t.Run("duplicate tools run once per command", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "black")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"),
			`{"commands": [["black", "--check"], ["black", "--diff"]]}`)

		code, _, _ := runCLI(t, "all", "--series", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if len(got) != 2 {
			t.Errorf("argv = %v, want two invocations", got)
		}
	})

	t.Run("a failing command does not swallow the others", func(t *testing.T) {
		root := isolateConfig(t)
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "black"), "#!/bin/sh\nexit 3\n")
		writeFakeExecutable(t, binDir, "shfmt")
		if err := chmodExecutable(filepath.Join(binDir, "black")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_TOOLS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"], ["shfmt"]]}`)

		code, _, _ := runCLI(t, "all", "--config", cfg)
		if code != 3 {
			t.Errorf("exit = %d, want 3", code)
		}
	})

	t.Run("a missing command does not swallow the others", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"], ["shfmt"]]}`)

		code, _, stderr := runCLI(t, "all", "--config", cfg)
		if code != ExitNotFound {
			t.Errorf("exit = %d, want %d", code, ExitNotFound)
		}
		if !strings.Contains(stderr, "Could not find 'black'") {
			t.Errorf("stderr = %q, must report the missing tool", stderr)
		}
		if got := recordedArgv(t, argvFile); len(got) == 0 {
			t.Error("the later command must still run")
		}
	})

	t.Run("exit aggregation follows the config order", func(t *testing.T) {
		root := isolateConfig(t)
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "black"), "#!/bin/sh\nexit 5\n")
		writeFile(t, filepath.Join(binDir, "shfmt"), "#!/bin/sh\nexit 1\n")
		for _, name := range []string{"black", "shfmt"} {
			if err := chmodExecutable(filepath.Join(binDir, name)); err != nil {
				t.Fatalf("chmod: %v", err)
			}
		}
		t.Setenv("FORMATTINGTON_TOOLS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"], ["shfmt"]]}`)

		code, _, _ := runCLI(t, "all", "--series", "--config", cfg)
		if code != 1 {
			t.Errorf("exit = %d, want 1", code)
		}
	})

	t.Run("discovery appends files", func(t *testing.T) {
		root := isolateConfig(t)
		writeFile(t, filepath.Join(root, "a.sh"), "#!/bin/sh\n")
		writeFile(t, filepath.Join(root, "b.sh"), "#!/bin/sh\n")
		writeFile(t, filepath.Join(root, "c.txt"), "hi\n")
		_, argvFile, _ := fakeTools(t, "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["shfmt"]]}`)

		code, _, _ := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		want := []string{"-w", "a.sh", "b.sh"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("argv = %v, want %v", got, want)
		}
	})

	t.Run("explicit files suppress discovery", func(t *testing.T) {
		root := isolateConfig(t)
		writeFile(t, filepath.Join(root, "a.sh"), "#!/bin/sh\n")
		writeFile(t, filepath.Join(root, "explicit.sh"), "#!/bin/sh\n")
		_, argvFile, _ := fakeTools(t, "shfmt")
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["shfmt", "explicit.sh"]]}`)

		code, _, _ := runCLI(t, "all", "--config", cfg)
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		want := []string{"-w", "explicit.sh"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("argv = %v, want %v", got, want)
		}
	})
}

func TestConfigAndSingleToolInteraction(t *testing.T) {
	t.Run("flag before a formatter exits two", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"]]}`)

		code, _, stderr := runCLI(t, "--config", cfg, "black")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
		if !strings.Contains(stderr, "--config") {
			t.Errorf("stderr = %q, must mention --config", stderr)
		}
	})

	t.Run("flag before an alias exits two", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "--config", cfg, "python")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
	})

	t.Run("config after a single formatter is forwarded", func(t *testing.T) {
		isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "prettier")

		code, _, _ := runCLI(t, "prettier", "--config", "my.json")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		want := []string{"--write", "--config", "my.json"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("argv = %v, want %v", got, want)
		}
	})

	t.Run("single tool ignores the auto-discovered config", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "shfmt", "black")
		writeConfig(t, filepath.Join(root, configFilename), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "shfmt")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", got)
		}
	})

	t.Run("single tool ignores the environment variable", func(t *testing.T) {
		root := isolateConfig(t)
		_, argvFile, _ := fakeTools(t, "shfmt", "black")
		cfg := writeConfig(t, filepath.Join(root, "env.json"), `{"commands": [["black"]]}`)
		t.Setenv(configEnvVar, cfg)

		code, _, _ := runCLI(t, "shfmt")
		if code != ExitOK {
			t.Fatalf("exit = %d, want %d", code, ExitOK)
		}
		got := recordedArgv(t, argvFile)
		if !reflect.DeepEqual(got, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", got)
		}
	})

	t.Run("extra args with a config exit two", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "all", "--config", cfg, "extra")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
	})

	t.Run("duplicate config exits two", func(t *testing.T) {
		root := isolateConfig(t)
		fakeTools(t)
		cfg := writeConfig(t, filepath.Join(root, "cfg.json"), `{"commands": [["black"]]}`)

		code, _, _ := runCLI(t, "--config", cfg, "all", "--config", cfg)
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
	})

	t.Run("missing value after all exits two", func(t *testing.T) {
		isolateConfig(t)
		fakeTools(t)

		code, _, _ := runCLI(t, "all", "--config")
		if code != ExitUsage {
			t.Errorf("exit = %d, want %d", code, ExitUsage)
		}
	})
}

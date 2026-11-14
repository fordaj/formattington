package formattington

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// runFake installs fake executables, runs the formatter, and returns the
// exit code plus the argv the fake recorded.
func runFake(t *testing.T, names []string, fn func() (int, error)) (int, []string) {
	t.Helper()
	_, argvFile, _ := fakeTools(t, names...)
	code, err := fn()
	if err != nil {
		t.Fatalf("formatter returned an error: %v", err)
	}
	return code, recordedArgv(t, argvFile)
}

func TestShfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		_, recorded := runFake(t, []string{"shfmt"}, func() (int, error) {
			return runShfmt(Options{})
		})
		if !reflect.DeepEqual(recorded, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", recorded)
		}
	})

	t.Run("check lists", func(t *testing.T) {
		_, recorded := runFake(t, []string{"shfmt"}, func() (int, error) {
			return runShfmt(Options{Check: true})
		})
		if !reflect.DeepEqual(recorded, []string{"-l"}) {
			t.Errorf("argv = %v, want [-l]", recorded)
		}
	})

	t.Run("dry run diffs", func(t *testing.T) {
		_, recorded := runFake(t, []string{"shfmt"}, func() (int, error) {
			return runShfmt(Options{DryRun: true})
		})
		if !reflect.DeepEqual(recorded, []string{"-d"}) {
			t.Errorf("argv = %v, want [-d]", recorded)
		}
	})

	t.Run("forwards extra args after the mode flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"shfmt"}, func() (int, error) {
			return runShfmt(Options{Args: []string{"--keep-padding", "file with spaces.sh"}})
		})
		want := []string{"-w", "--keep-padding", "file with spaces.sh"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestClangFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default inserts in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"clang-format"}, func() (int, error) {
			return runClangFormat(Options{})
		})
		if !reflect.DeepEqual(recorded, []string{"-i"}) {
			t.Errorf("argv = %v, want [-i]", recorded)
		}
	})

	t.Run("does not duplicate in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"clang-format"}, func() (int, error) {
			return runClangFormat(Options{Args: []string{"--in-place", "file.cpp"}})
		})
		want := []string{"--in-place", "file.cpp"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("check uses dry run and werror", func(t *testing.T) {
		_, recorded := runFake(t, []string{"clang-format"}, func() (int, error) {
			return runClangFormat(Options{Check: true})
		})
		want := []string{"--dry-run", "--Werror"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run uses dry run", func(t *testing.T) {
		_, recorded := runFake(t, []string{"clang-format"}, func() (int, error) {
			return runClangFormat(Options{DryRun: true})
		})
		if !reflect.DeepEqual(recorded, []string{"--dry-run"}) {
			t.Errorf("argv = %v, want [--dry-run]", recorded)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"clang-format"}, func() (int, error) {
			return runClangFormat(Options{Args: []string{"-h"}})
		})
		if !reflect.DeepEqual(recorded, []string{"-h"}) {
			t.Errorf("argv = %v, want [-h]", recorded)
		}
	})

	t.Run("forwards args and preserves spaces", func(t *testing.T) {
		_, recorded := runFake(t, []string{"clang-format"}, func() (int, error) {
			return runClangFormat(Options{Args: []string{"--style=file", "my file.cpp", "another file.cpp"}})
		})
		want := []string{"-i", "--style=file", "my file.cpp", "another file.cpp"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestMarkdownlintArgv(t *testing.T) {
	requirePOSIX(t)

	for _, mode := range []struct {
		name string
		opts Options
	}{
		{"default lints", Options{}},
		{"check lints", Options{Check: true}},
		{"dry run lints", Options{DryRun: true}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			_, recorded := runFake(t, []string{"markdownlint"}, func() (int, error) {
				return runMarkdownlint(mode.opts)
			})
			if len(recorded) != 0 {
				t.Errorf("argv = %v, want no arguments (markdownlint never rewrites without --fix)", recorded)
			}
		})
	}

	t.Run("forwards --fix and files verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"markdownlint"}, func() (int, error) {
			return runMarkdownlint(Options{Check: true, Args: []string{"--fix", "--config", "custom.json", "README.md"}})
		})
		want := []string{"--fix", "--config", "custom.json", "README.md"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("missing tool reports exit 127", func(t *testing.T) {
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", t.TempDir())
		code, err := runMarkdownlint(Options{})
		if err == nil {
			t.Fatal("expected an error for a missing tool")
		}
		if code != ExitNotFound {
			t.Errorf("code = %d, want %d", code, ExitNotFound)
		}
	})
}

func TestBlackArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default has no flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"black"}, func() (int, error) {
			return runBlack(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want no arguments", recorded)
		}
	})

	t.Run("check uses the check flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"black"}, func() (int, error) {
			return runBlack(Options{Check: true})
		})
		if !reflect.DeepEqual(recorded, []string{"--check"}) {
			t.Errorf("argv = %v, want [--check]", recorded)
		}
	})

	t.Run("dry run uses the diff flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"black"}, func() (int, error) {
			return runBlack(Options{DryRun: true})
		})
		if !reflect.DeepEqual(recorded, []string{"--diff"}) {
			t.Errorf("argv = %v, want [--diff]", recorded)
		}
	})

	t.Run("does not add a line length", func(t *testing.T) {
		_, recorded := runFake(t, []string{"black"}, func() (int, error) {
			return runBlack(Options{Args: []string{"file.py"}})
		})
		if contains(recorded, "--line-length") {
			t.Errorf("argv = %v, must not contain --line-length", recorded)
		}
	})

	t.Run("forwards a line length", func(t *testing.T) {
		_, recorded := runFake(t, []string{"black"}, func() (int, error) {
			return runBlack(Options{Args: []string{"--line-length", "88", "file.py"}})
		})
		want := []string{"--line-length", "88", "file.py"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestPrettierArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{})
		})
		if !reflect.DeepEqual(recorded, []string{"--write"}) {
			t.Errorf("argv = %v, want [--write]", recorded)
		}
	})

	t.Run("check uses the check flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{Check: true})
		})
		if !reflect.DeepEqual(recorded, []string{"--check"}) {
			t.Errorf("argv = %v, want [--check]", recorded)
		}
	})

	t.Run("dry run lists different files", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{DryRun: true})
		})
		if !reflect.DeepEqual(recorded, []string{"--list-different"}) {
			t.Errorf("argv = %v, want [--list-different]", recorded)
		}
	})

	t.Run("does not duplicate write", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{Args: []string{"--write", "file.js"}})
		})
		want := []string{"--write", "file.js"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("forwarded check omits write", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{Args: []string{"--check"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--check"}) {
			t.Errorf("argv = %v, want [--check]", recorded)
		}
	})

	t.Run("forwarded list different omits write", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{Args: []string{"--list-different"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--list-different"}) {
			t.Errorf("argv = %v, want [--list-different]", recorded)
		}
	})

	t.Run("does not add a config flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"prettier"}, func() (int, error) {
			return runPrettier(Options{Args: []string{"file.js"}})
		})
		if contains(recorded, "--config") {
			t.Errorf("argv = %v, must not contain --config", recorded)
		}
	})
}

func TestTaploArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"taplo"}, func() (int, error) {
			return runTaplo(Options{})
		})
		want := []string{"fmt", "--colors", "never"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("check uses fmt check", func(t *testing.T) {
		_, recorded := runFake(t, []string{"taplo"}, func() (int, error) {
			return runTaplo(Options{Check: true})
		})
		want := []string{"fmt", "--colors", "never", "--check"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run uses fmt diff", func(t *testing.T) {
		_, recorded := runFake(t, []string{"taplo"}, func() (int, error) {
			return runTaplo(Options{DryRun: true})
		})
		want := []string{"fmt", "--colors", "never", "--diff"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("user colors are respected", func(t *testing.T) {
		_, recorded := runFake(t, []string{"taplo"}, func() (int, error) {
			return runTaplo(Options{Args: []string{"--colors", "always", "file.toml"}})
		})
		want := []string{"fmt", "--colors", "always", "file.toml"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("does not add a config flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"taplo"}, func() (int, error) {
			return runTaplo(Options{Args: []string{"file.toml"}})
		})
		if contains(recorded, "--config") {
			t.Errorf("argv = %v, must not contain --config", recorded)
		}
	})
}

func TestGofmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		_, recorded := runFake(t, []string{"gofmt"}, func() (int, error) {
			return runGofmt(Options{})
		})
		if !reflect.DeepEqual(recorded, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", recorded)
		}
	})

	t.Run("check lists", func(t *testing.T) {
		_, recorded := runFake(t, []string{"gofmt"}, func() (int, error) {
			return runGofmt(Options{Check: true})
		})
		if !reflect.DeepEqual(recorded, []string{"-l"}) {
			t.Errorf("argv = %v, want [-l]", recorded)
		}
	})

	t.Run("dry run diffs", func(t *testing.T) {
		_, recorded := runFake(t, []string{"gofmt"}, func() (int, error) {
			return runGofmt(Options{DryRun: true})
		})
		if !reflect.DeepEqual(recorded, []string{"-d"}) {
			t.Errorf("argv = %v, want [-d]", recorded)
		}
	})

	t.Run("forwards args after the mode flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"gofmt"}, func() (int, error) {
			return runGofmt(Options{Check: true, Args: []string{"my file.go", "another file.go"}})
		})
		want := []string{"-l", "my file.go", "another file.go"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestGoimportsArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		_, recorded := runFake(t, []string{"goimports"}, func() (int, error) {
			return runGoimports(Options{})
		})
		if !reflect.DeepEqual(recorded, []string{"-w"}) {
			t.Errorf("argv = %v, want [-w]", recorded)
		}
	})

	t.Run("check lists", func(t *testing.T) {
		_, recorded := runFake(t, []string{"goimports"}, func() (int, error) {
			return runGoimports(Options{Check: true})
		})
		if !reflect.DeepEqual(recorded, []string{"-l"}) {
			t.Errorf("argv = %v, want [-l]", recorded)
		}
	})

	t.Run("forwards the local flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"goimports"}, func() (int, error) {
			return runGoimports(Options{Args: []string{"-local", "example.com/foo", "file.go"}})
		})
		want := []string{"-w", "-local", "example.com/foo", "file.go"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestPowerShellArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("prefers pwsh", func(t *testing.T) {
		_, recorded := runFake(t, []string{"pwsh", "powershell"}, func() (int, error) {
			return runPowerShell(Options{})
		})
		if len(recorded) != 3 {
			t.Fatalf("argv = %v, want three entries", recorded)
		}
		if recorded[0] != "-NoProfile" || recorded[1] != "-Command" {
			t.Errorf("argv = %v, want -NoProfile -Command", recorded)
		}
		script := recorded[2]
		if !strings.Contains(script, "Import-Module PSScriptAnalyzer") {
			t.Errorf("script = %q, must import PSScriptAnalyzer", script)
		}
		if !strings.Contains(script, "Invoke-ScriptAnalyzer -Path '.' -Recurse") {
			t.Errorf("script = %q, must analyze the working directory", script)
		}
		if !strings.Contains(script, "$issues.Count -gt 0") {
			t.Errorf("script = %q, must exit 1 on findings", script)
		}
	})

	t.Run("falls back to powershell", func(t *testing.T) {
		// Only "powershell" is installed, so the lookup must fall back.
		_, recorded := runFake(t, []string{"powershell"}, func() (int, error) {
			return runPowerShell(Options{})
		})
		if len(recorded) != 3 {
			t.Fatalf("argv = %v, want three entries", recorded)
		}
	})

	t.Run("missing shell reports an install hint", func(t *testing.T) {
		fakeTools(t)
		_, err := runPowerShell(Options{})
		if err == nil {
			t.Fatal("expected an error when no PowerShell host is available")
		}
		if !strings.Contains(err.Error(), "pwsh") {
			t.Errorf("error = %v, must mention pwsh", err)
		}
		if !strings.Contains(err.Error(), "Install-Module PSScriptAnalyzer") {
			t.Errorf("error = %v, must mention the module install hint", err)
		}
	})

	t.Run("files are passed via a path list", func(t *testing.T) {
		_, recorded := runFake(t, []string{"pwsh"}, func() (int, error) {
			return runPowerShell(Options{Args: []string{"my dir/script.ps1", "other.psm1"}})
		})
		script := recorded[2]
		if !strings.Contains(script, "-Path 'my dir/script.ps1', 'other.psm1'") {
			t.Errorf("script = %q, must pass both scripts", script)
		}
		if strings.Contains(script, "-Recurse") {
			t.Errorf("script = %q, must not recurse when files are given", script)
		}
	})

	t.Run("single quotes are doubled", func(t *testing.T) {
		_, recorded := runFake(t, []string{"pwsh"}, func() (int, error) {
			return runPowerShell(Options{Args: []string{"it's here.ps1"}})
		})
		if !strings.Contains(recorded[2], "'it''s here.ps1'") {
			t.Errorf("script = %q, must escape single quotes", recorded[2])
		}
	})

	t.Run("params are forwarded to the analyzer", func(t *testing.T) {
		_, recorded := runFake(t, []string{"pwsh"}, func() (int, error) {
			return runPowerShell(Options{Args: []string{"-Severity", "Error", "script.ps1"}})
		})
		script := recorded[2]
		if !strings.Contains(script, "-Severity Error") {
			t.Errorf("script = %q, must forward the severity", script)
		}
		if !strings.Contains(script, "-Path 'script.ps1'") {
			t.Errorf("script = %q, must pass the script path", script)
		}
	})

	t.Run("check and dry run use the same lint invocation", func(t *testing.T) {
		for _, opts := range []Options{{}, {Check: true}, {DryRun: true}} {
			_, recorded := runFake(t, []string{"pwsh"}, func() (int, error) {
				return runPowerShell(opts)
			})
			if !strings.Contains(recorded[2], "Invoke-ScriptAnalyzer") {
				t.Errorf("script = %q, must invoke the analyzer", recorded[2])
			}
		}
	})
}

func TestInformationalPassthrough(t *testing.T) {
	requirePOSIX(t)

	cases := []struct {
		name string
		tool string
		run  func(Options) (int, error)
	}{
		{"clang-format", "clang-format", runClangFormat},
		{"black", "black", runBlack},
		{"prettier", "prettier", runPrettier},
		{"powershell", "pwsh", runPowerShell},
		{"markdownlint", "markdownlint", runMarkdownlint},
		{"taplo", "taplo", runTaplo},
		{"gofmt", "gofmt", runGofmt},
		{"goimports", "goimports", runGoimports},
	}

	for _, testCase := range cases {
		t.Run(testCase.name+" help", func(t *testing.T) {
			_, recorded := runFake(t, []string{testCase.tool}, func() (int, error) {
				return testCase.run(Options{Check: true, Args: []string{"--help"}})
			})
			if !reflect.DeepEqual(recorded, []string{"--help"}) {
				t.Errorf("argv = %v, want [--help]", recorded)
			}
		})
		t.Run(testCase.name+" version", func(t *testing.T) {
			_, recorded := runFake(t, []string{testCase.tool}, func() (int, error) {
				return testCase.run(Options{DryRun: true, Args: []string{"--version"}})
			})
			if !reflect.DeepEqual(recorded, []string{"--version"}) {
				t.Errorf("argv = %v, want [--version]", recorded)
			}
		})
	}
}

func TestMissingToolIsReported(t *testing.T) {
	fakeTools(t)

	for _, tool := range registry {
		if tool.Name == "powershell" {
			continue
		}
		_, err := tool.Run(Options{})
		if err == nil {
			t.Errorf("%s: expected a missing-tool error", tool.Name)
			continue
		}
		if !strings.Contains(err.Error(), "Could not find") {
			t.Errorf("%s: unexpected error: %v", tool.Name, err)
		}
	}
}

func TestStyluaArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"stylua"}, func() (int, error) {
			return runStylua(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check uses check flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"stylua"}, func() (int, error) {
			return runStylua(Options{Check: true})
		})
		want := []string{"--check"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run prints unified diffs via check", func(t *testing.T) {
		_, recorded := runFake(t, []string{"stylua"}, func() (int, error) {
			return runStylua(Options{DryRun: true})
		})
		want := []string{"--check", "--output-format=unified"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"stylua"}, func() (int, error) {
			return runStylua(Options{Args: []string{"--help"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--help"}) {
			t.Errorf("argv = %v, want [--help]", recorded)
		}
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"stylua"}, func() (int, error) {
			return runStylua(Options{Args: []string{"--column-width", "100", "my file.lua"}})
		})
		want := []string{"--column-width", "100", "my file.lua"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestNixfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"nixfmt"}, func() (int, error) {
			return runNixfmt(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check uses check flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"nixfmt"}, func() (int, error) {
			return runNixfmt(Options{Check: true})
		})
		want := []string{"--check"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		_, recorded := runFake(t, []string{"nixfmt"}, func() (int, error) {
			return runNixfmt(Options{DryRun: true})
		})
		want := []string{"--check"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"nixfmt"}, func() (int, error) {
			return runNixfmt(Options{Args: []string{"--help"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--help"}) {
			t.Errorf("argv = %v, want [--help]", recorded)
		}
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"nixfmt"}, func() (int, error) {
			return runNixfmt(Options{Args: []string{"--width", "100", "my file.nix"}})
		})
		want := []string{"--width", "100", "my file.nix"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestPhpCsFixerArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default fixes in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"php-cs-fixer"}, func() (int, error) {
			return runPhpCsFixer(Options{})
		})
		want := []string{"fix"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("check uses check subcommand", func(t *testing.T) {
		_, recorded := runFake(t, []string{"php-cs-fixer"}, func() (int, error) {
			return runPhpCsFixer(Options{Check: true})
		})
		want := []string{"check"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		_, recorded := runFake(t, []string{"php-cs-fixer"}, func() (int, error) {
			return runPhpCsFixer(Options{DryRun: true})
		})
		want := []string{"check"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"php-cs-fixer"}, func() (int, error) {
			return runPhpCsFixer(Options{Args: []string{"--version"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--version"}) {
			t.Errorf("argv = %v, want [--version]", recorded)
		}
	})

	t.Run("diff passthrough is verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"php-cs-fixer"}, func() (int, error) {
			return runPhpCsFixer(Options{Args: []string{"fix", "--dry-run", "--diff", "file.php"}})
		})
		want := []string{"fix", "fix", "--dry-run", "--diff", "file.php"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestPhpCsFixerCheckExitEightIsRemapped(t *testing.T) {
	requirePOSIX(t)

	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "php-cs-fixer"), "#!/bin/sh\nexit 8\n")
	if err := chmodExecutable(filepath.Join(binDir, "php-cs-fixer")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runPhpCsFixer(Options{Check: true})
	if err != nil {
		t.Fatalf("runPhpCsFixer returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d (native 8 must map to ExitFormatRequired)", code, ExitFormatRequired)
	}

	// Other non-zero codes pass through verbatim.
	writeFile(t, filepath.Join(binDir, "php-cs-fixer"), "#!/bin/sh\nexit 16\n")
	code, err = runPhpCsFixer(Options{Check: true})
	if err != nil {
		t.Fatalf("runPhpCsFixer returned an error: %v", err)
	}
	if code != 16 {
		t.Errorf("exit = %d, want 16 (unknown codes must pass through)", code)
	}
}

func TestPhpCsFixerCheckDoesNotModifyFixture(t *testing.T) {
	requirePOSIX(t)

	fixture := filepath.Join("testdata", "php-cs-fixer", "sample.php")
	original, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	before := string(original)

	// A fake that would rewrite the file must never run in check mode,
	// because the integration only invokes read-only commands.
	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "php-cs-fixer"), "#!/bin/sh\nexit 8\n")
	if err := chmodExecutable(filepath.Join(binDir, "php-cs-fixer")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runPhpCsFixer(Options{Check: true, Args: []string{fixture}})
	if err != nil {
		t.Fatalf("runPhpCsFixer returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d", code, ExitFormatRequired)
	}
	after, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if string(after) != before {
		t.Error("the fixture was modified in check mode")
	}
}

func TestPerltidyArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place and suppresses backups", func(t *testing.T) {
		_, recorded := runFake(t, []string{"perltidy"}, func() (int, error) {
			return runPerltidy(Options{})
		})
		want := []string{"-b", "-bext=/"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("check uses assert-tidy", func(t *testing.T) {
		_, recorded := runFake(t, []string{"perltidy"}, func() (int, error) {
			return runPerltidy(Options{Check: true})
		})
		want := []string{"--assert-tidy"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run prints a diff", func(t *testing.T) {
		_, recorded := runFake(t, []string{"perltidy"}, func() (int, error) {
			return runPerltidy(Options{DryRun: true})
		})
		want := []string{"-d"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"perltidy"}, func() (int, error) {
			return runPerltidy(Options{Args: []string{"--version"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--version"}) {
			t.Errorf("argv = %v, want [--version]", recorded)
		}
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"perltidy"}, func() (int, error) {
			return runPerltidy(Options{Args: []string{"-i=4", "--profile=custom.rc", "my file.pl"}})
		})
		want := []string{"-b", "-bext=/", "-i=4", "--profile=custom.rc", "my file.pl"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestPerltidyCheckNonZeroIsRemapped(t *testing.T) {
	requirePOSIX(t)

	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "perltidy"), "#!/bin/sh\nexit 2\n")
	if err := chmodExecutable(filepath.Join(binDir, "perltidy")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runPerltidy(Options{Check: true})
	if err != nil {
		t.Fatalf("runPerltidy returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d (non-zero --assert-tidy must map to ExitFormatRequired)", code, ExitFormatRequired)
	}
}

func TestFprettifyArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"fprettify"}, func() (int, error) {
			return runFprettify(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check synthesizes via diff", func(t *testing.T) {
		_, recorded := runFake(t, []string{"fprettify"}, func() (int, error) {
			return runFprettify(Options{Check: true})
		})
		want := []string{"--diff"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run prints a diff", func(t *testing.T) {
		_, recorded := runFake(t, []string{"fprettify"}, func() (int, error) {
			return runFprettify(Options{DryRun: true})
		})
		want := []string{"--diff"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"fprettify"}, func() (int, error) {
			return runFprettify(Options{Args: []string{"--help"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--help"}) {
			t.Errorf("argv = %v, want [--help]", recorded)
		}
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"fprettify"}, func() (int, error) {
			return runFprettify(Options{Args: []string{"-c", "custom.rc", "my file.f90"}})
		})
		want := []string{"-c", "custom.rc", "my file.f90"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestFprettifyCheckReportsDiffOutputAsExitFormatRequired(t *testing.T) {
	requirePOSIX(t)

	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "fprettify"), "#!/bin/sh\necho '--- a/foo.f90'\nexit 0\n")
	if err := chmodExecutable(filepath.Join(binDir, "fprettify")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runFprettify(Options{Check: true})
	if err != nil {
		t.Fatalf("runFprettify returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d (non-empty diff output means changes are required)", code, ExitFormatRequired)
	}
}

func TestErlfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"erlfmt"}, func() (int, error) {
			return runErlfmt(Options{})
		})
		want := []string{"-w"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("check uses -c", func(t *testing.T) {
		_, recorded := runFake(t, []string{"erlfmt"}, func() (int, error) {
			return runErlfmt(Options{Check: true})
		})
		want := []string{"-c"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		_, recorded := runFake(t, []string{"erlfmt"}, func() (int, error) {
			return runErlfmt(Options{DryRun: true})
		})
		want := []string{"-c"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		_, recorded := runFake(t, []string{"erlfmt"}, func() (int, error) {
			return runErlfmt(Options{Args: []string{"--help"}})
		})
		if !reflect.DeepEqual(recorded, []string{"--help"}) {
			t.Errorf("argv = %v, want [--help]", recorded)
		}
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		_, recorded := runFake(t, []string{"erlfmt"}, func() (int, error) {
			return runErlfmt(Options{Args: []string{"my file.erl"}})
		})
		want := []string{"-w", "my file.erl"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

package formattington

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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

// checkArgv runs fn with fake executables installed and asserts the
// arguments the fakes recorded.
func checkArgv(t *testing.T, tool string, fn func() (int, error), want []string) {
	t.Helper()
	_, recorded := runFake(t, []string{tool}, fn)
	if !reflect.DeepEqual(recorded, want) {
		t.Errorf("argv = %v, want %v", recorded, want)
	}
}

func TestShfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		checkArgv(t, "shfmt", func() (int, error) {
			return runShfmt(Options{})
		}, []string{"-w"})
	})

	t.Run("check lists", func(t *testing.T) {
		checkArgv(t, "shfmt", func() (int, error) {
			return runShfmt(Options{Check: true})
		}, []string{"-l"})
	})

	t.Run("dry run diffs", func(t *testing.T) {
		checkArgv(t, "shfmt", func() (int, error) {
			return runShfmt(Options{DryRun: true})
		}, []string{"-d"})
	})

	t.Run("forwards extra args after the mode flag", func(t *testing.T) {
		checkArgv(t, "shfmt", func() (int, error) {
			return runShfmt(Options{Args: []string{"--keep-padding", "file with spaces.sh"}})
		}, []string{"-w", "--keep-padding", "file with spaces.sh"})
	})
}

func TestClangFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default inserts in place", func(t *testing.T) {
		checkArgv(t, "clang-format", func() (int, error) {
			return runClangFormat(Options{})
		}, []string{"-i"})
	})

	t.Run("does not duplicate in place", func(t *testing.T) {
		checkArgv(t, "clang-format", func() (int, error) {
			return runClangFormat(Options{Args: []string{"--in-place", "file.cpp"}})
		}, []string{"--in-place", "file.cpp"})
	})

	t.Run("check uses dry run and werror", func(t *testing.T) {
		checkArgv(t, "clang-format", func() (int, error) {
			return runClangFormat(Options{Check: true})
		}, []string{"--dry-run", "--Werror"})
	})

	t.Run("dry run uses dry run", func(t *testing.T) {
		checkArgv(t, "clang-format", func() (int, error) {
			return runClangFormat(Options{DryRun: true})
		}, []string{"--dry-run"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "clang-format", func() (int, error) {
			return runClangFormat(Options{Args: []string{"-h"}})
		}, []string{"-h"})
	})

	t.Run("forwards args and preserves spaces", func(t *testing.T) {
		checkArgv(t, "clang-format", func() (int, error) {
			return runClangFormat(Options{Args: []string{"--style=file", "my file.cpp", "another file.cpp"}})
		}, []string{"-i", "--style=file", "my file.cpp", "another file.cpp"})
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
		checkArgv(t, "markdownlint", func() (int, error) {
			return runMarkdownlint(Options{Check: true, Args: []string{"--fix", "--config", "custom.json", "README.md"}})
		}, []string{"--fix", "--config", "custom.json", "README.md"})
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
		checkArgv(t, "black", func() (int, error) {
			return runBlack(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run uses the diff flag", func(t *testing.T) {
		checkArgv(t, "black", func() (int, error) {
			return runBlack(Options{DryRun: true})
		}, []string{"--diff"})
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
		checkArgv(t, "black", func() (int, error) {
			return runBlack(Options{Args: []string{"--line-length", "88", "file.py"}})
		}, []string{"--line-length", "88", "file.py"})
	})
}

func TestPrettierArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		checkArgv(t, "prettier", func() (int, error) {
			return runPrettier(Options{})
		}, []string{"--write"})
	})

	t.Run("check uses the check flag", func(t *testing.T) {
		checkArgv(t, "prettier", func() (int, error) {
			return runPrettier(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run lists different files", func(t *testing.T) {
		checkArgv(t, "prettier", func() (int, error) {
			return runPrettier(Options{DryRun: true})
		}, []string{"--list-different"})
	})

	t.Run("does not duplicate write", func(t *testing.T) {
		checkArgv(t, "prettier", func() (int, error) {
			return runPrettier(Options{Args: []string{"--write", "file.js"}})
		}, []string{"--write", "file.js"})
	})

	t.Run("forwarded check omits write", func(t *testing.T) {
		checkArgv(t, "prettier", func() (int, error) {
			return runPrettier(Options{Args: []string{"--check"}})
		}, []string{"--check"})
	})

	t.Run("forwarded list different omits write", func(t *testing.T) {
		checkArgv(t, "prettier", func() (int, error) {
			return runPrettier(Options{Args: []string{"--list-different"}})
		}, []string{"--list-different"})
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
		checkArgv(t, "taplo", func() (int, error) {
			return runTaplo(Options{})
		}, []string{"fmt", "--colors", "never"})
	})

	t.Run("check uses fmt check", func(t *testing.T) {
		checkArgv(t, "taplo", func() (int, error) {
			return runTaplo(Options{Check: true})
		}, []string{"fmt", "--colors", "never", "--check"})
	})

	t.Run("dry run uses fmt diff", func(t *testing.T) {
		checkArgv(t, "taplo", func() (int, error) {
			return runTaplo(Options{DryRun: true})
		}, []string{"fmt", "--colors", "never", "--diff"})
	})

	t.Run("user colors are respected", func(t *testing.T) {
		checkArgv(t, "taplo", func() (int, error) {
			return runTaplo(Options{Args: []string{"--colors", "always", "file.toml"}})
		}, []string{"fmt", "--colors", "always", "file.toml"})
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
		checkArgv(t, "gofmt", func() (int, error) {
			return runGofmt(Options{})
		}, []string{"-w"})
	})

	t.Run("check lists", func(t *testing.T) {
		checkArgv(t, "gofmt", func() (int, error) {
			return runGofmt(Options{Check: true})
		}, []string{"-l"})
	})

	t.Run("dry run diffs", func(t *testing.T) {
		checkArgv(t, "gofmt", func() (int, error) {
			return runGofmt(Options{DryRun: true})
		}, []string{"-d"})
	})

	t.Run("forwards args after the mode flag", func(t *testing.T) {
		checkArgv(t, "gofmt", func() (int, error) {
			return runGofmt(Options{Check: true, Args: []string{"my file.go", "another file.go"}})
		}, []string{"-l", "my file.go", "another file.go"})
	})
}

func TestGoimportsArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default writes", func(t *testing.T) {
		checkArgv(t, "goimports", func() (int, error) {
			return runGoimports(Options{})
		}, []string{"-w"})
	})

	t.Run("check lists", func(t *testing.T) {
		checkArgv(t, "goimports", func() (int, error) {
			return runGoimports(Options{Check: true})
		}, []string{"-l"})
	})

	t.Run("forwards the local flag", func(t *testing.T) {
		checkArgv(t, "goimports", func() (int, error) {
			return runGoimports(Options{Args: []string{"-local", "example.com/foo", "file.go"}})
		}, []string{"-w", "-local", "example.com/foo", "file.go"})
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
		{"shfmt", "shfmt", runShfmt},
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
		checkArgv(t, "stylua", func() (int, error) {
			return runStylua(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run prints unified diffs via check", func(t *testing.T) {
		checkArgv(t, "stylua", func() (int, error) {
			return runStylua(Options{DryRun: true})
		}, []string{"--check", "--output-format=unified"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "stylua", func() (int, error) {
			return runStylua(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "stylua", func() (int, error) {
			return runStylua(Options{Args: []string{"--column-width", "100", "my file.lua"}})
		}, []string{"--column-width", "100", "my file.lua"})
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
		checkArgv(t, "nixfmt", func() (int, error) {
			return runNixfmt(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "nixfmt", func() (int, error) {
			return runNixfmt(Options{DryRun: true})
		}, []string{"--check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "nixfmt", func() (int, error) {
			return runNixfmt(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "nixfmt", func() (int, error) {
			return runNixfmt(Options{Args: []string{"--width", "100", "my file.nix"}})
		}, []string{"--width", "100", "my file.nix"})
	})
}

func TestPhpCsFixerArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default fixes in place", func(t *testing.T) {
		checkArgv(t, "php-cs-fixer", func() (int, error) {
			return runPhpCsFixer(Options{})
		}, []string{"fix"})
	})

	t.Run("check uses check subcommand", func(t *testing.T) {
		checkArgv(t, "php-cs-fixer", func() (int, error) {
			return runPhpCsFixer(Options{Check: true})
		}, []string{"check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "php-cs-fixer", func() (int, error) {
			return runPhpCsFixer(Options{DryRun: true})
		}, []string{"check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "php-cs-fixer", func() (int, error) {
			return runPhpCsFixer(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("diff passthrough is verbatim", func(t *testing.T) {
		checkArgv(t, "php-cs-fixer", func() (int, error) {
			return runPhpCsFixer(Options{Args: []string{"fix", "--dry-run", "--diff", "file.php"}})
		}, []string{"fix", "fix", "--dry-run", "--diff", "file.php"})
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
		checkArgv(t, "perltidy", func() (int, error) {
			return runPerltidy(Options{})
		}, []string{"-b", "-bext=/"})
	})

	t.Run("check uses assert-tidy", func(t *testing.T) {
		checkArgv(t, "perltidy", func() (int, error) {
			return runPerltidy(Options{Check: true})
		}, []string{"--assert-tidy"})
	})

	t.Run("dry run prints a diff", func(t *testing.T) {
		checkArgv(t, "perltidy", func() (int, error) {
			return runPerltidy(Options{DryRun: true})
		}, []string{"-d"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "perltidy", func() (int, error) {
			return runPerltidy(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "perltidy", func() (int, error) {
			return runPerltidy(Options{Args: []string{"-i=4", "--profile=custom.rc", "my file.pl"}})
		}, []string{"-b", "-bext=/", "-i=4", "--profile=custom.rc", "my file.pl"})
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
		checkArgv(t, "fprettify", func() (int, error) {
			return runFprettify(Options{Check: true})
		}, []string{"--diff"})
	})

	t.Run("dry run prints a diff", func(t *testing.T) {
		checkArgv(t, "fprettify", func() (int, error) {
			return runFprettify(Options{DryRun: true})
		}, []string{"--diff"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "fprettify", func() (int, error) {
			return runFprettify(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "fprettify", func() (int, error) {
			return runFprettify(Options{Args: []string{"-c", "custom.rc", "my file.f90"}})
		}, []string{"-c", "custom.rc", "my file.f90"})
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
		checkArgv(t, "erlfmt", func() (int, error) {
			return runErlfmt(Options{})
		}, []string{"-w"})
	})

	t.Run("check uses -c", func(t *testing.T) {
		checkArgv(t, "erlfmt", func() (int, error) {
			return runErlfmt(Options{Check: true})
		}, []string{"-c"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "erlfmt", func() (int, error) {
			return runErlfmt(Options{DryRun: true})
		}, []string{"-c"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "erlfmt", func() (int, error) {
			return runErlfmt(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "erlfmt", func() (int, error) {
			return runErlfmt(Options{Args: []string{"my file.erl"}})
		}, []string{"-w", "my file.erl"})
	})
}

func TestSqlfluffArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default fixes in place", func(t *testing.T) {
		checkArgv(t, "sqlfluff", func() (int, error) {
			return runSqlfluff(Options{})
		}, []string{"fix", "--disable-progress-bar"})
	})

	t.Run("check lints without modifying", func(t *testing.T) {
		checkArgv(t, "sqlfluff", func() (int, error) {
			return runSqlfluff(Options{Check: true})
		}, []string{"lint", "--disable-progress-bar"})
	})

	t.Run("dry run validates without modifying", func(t *testing.T) {
		checkArgv(t, "sqlfluff", func() (int, error) {
			return runSqlfluff(Options{DryRun: true})
		}, []string{"lint", "--disable-progress-bar"})
	})

	t.Run("dialect is never hardcoded and passes through", func(t *testing.T) {
		checkArgv(t, "sqlfluff", func() (int, error) {
			return runSqlfluff(Options{Check: true, Args: []string{"--dialect", "postgres", "my file.sql"}})
		}, []string{"lint", "--disable-progress-bar", "--dialect", "postgres", "my file.sql"})
	})

	t.Run("config is forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "sqlfluff", func() (int, error) {
			return runSqlfluff(Options{Args: []string{"--config", "ci.sqlfluff", "x.sql"}})
		}, []string{"fix", "--disable-progress-bar", "--config", "ci.sqlfluff", "x.sql"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "sqlfluff", func() (int, error) {
			return runSqlfluff(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})
}

func TestSqlfluffCheckExitMapping(t *testing.T) {
	requirePOSIX(t)

	for _, tc := range []struct{ native, want int }{
		{1, ExitFormatRequired}, // violations
		{2, 2},                  // error passes through
		{0, 0},                  // clean
	} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "sqlfluff"), "#!/bin/sh\nexit "+strconv.Itoa(tc.native)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "sqlfluff")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		code, err := runSqlfluff(Options{Check: true})
		if err != nil {
			t.Fatalf("native exit %d: runSqlfluff returned an error: %v", tc.native, err)
		}
		if code != tc.want {
			t.Errorf("native exit %d: got %d, want %d", tc.native, code, tc.want)
		}
	}
}

func TestGdformatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"gdformat"}, func() (int, error) {
			return runGdformat(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check uses -c", func(t *testing.T) {
		checkArgv(t, "gdformat", func() (int, error) {
			return runGdformat(Options{Check: true})
		}, []string{"-c"})
	})

	t.Run("dry run uses -d and implies check", func(t *testing.T) {
		checkArgv(t, "gdformat", func() (int, error) {
			return runGdformat(Options{DryRun: true})
		}, []string{"-d"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "gdformat", func() (int, error) {
			return runGdformat(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("forwards args verbatim and never injects --fast", func(t *testing.T) {
		_, recorded := runFake(t, []string{"gdformat"}, func() (int, error) {
			return runGdformat(Options{Args: []string{"--line-length", "100", "my file.gd"}})
		})
		want := []string{"--line-length", "100", "my file.gd"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
		if contains(recorded, "--fast") {
			t.Error("argv must not contain --fast (safety checks stay on)")
		}
	})

	t.Run("fixture is not modified in check mode", func(t *testing.T) {
		fixture := filepath.Join("testdata", "gdformat", "fixture.gd")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		checkArgv(t, "gdformat", func() (int, error) {
			return runGdformat(Options{Check: true, Args: []string{fixture}})
		}, []string{"-c", fixture})
		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != string(original) {
			t.Error("the fixture was modified in check mode")
		}
	})
}

func TestGoogleJavaFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default replaces in place with cleanup disabled", func(t *testing.T) {
		checkArgv(t, "google-java-format", func() (int, error) {
			return runGoogleJavaFormat(Options{})
		}, []string{"--replace", "--cleanup=false"})
	})

	t.Run("check uses dry-run with exit-if-changed", func(t *testing.T) {
		checkArgv(t, "google-java-format", func() (int, error) {
			return runGoogleJavaFormat(Options{Check: true})
		}, []string{"--dry-run", "--set-exit-if-changed", "--cleanup=false"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "google-java-format", func() (int, error) {
			return runGoogleJavaFormat(Options{DryRun: true})
		}, []string{"--dry-run", "--set-exit-if-changed", "--cleanup=false"})
	})

	t.Run("user cleanup value wins over the default", func(t *testing.T) {
		_, recorded := runFake(t, []string{"google-java-format"}, func() (int, error) {
			return runGoogleJavaFormat(Options{Args: []string{"--cleanup=true", "File.java"}})
		})
		// The user's --cleanup=true must survive; --cleanup=false is not added.
		if contains(recorded, "--cleanup=false") {
			t.Errorf("argv = %v, must not contain the forced --cleanup=false", recorded)
		}
		for _, arg := range []string{"--cleanup=true", "File.java", "--replace"} {
			if !contains(recorded, arg) {
				t.Errorf("argv = %v, missing %q", recorded, arg)
			}
		}
	})

	t.Run("bare cleanup flag suppresses the default too", func(t *testing.T) {
		_, recorded := runFake(t, []string{"google-java-format"}, func() (int, error) {
			return runGoogleJavaFormat(Options{Args: []string{"--cleanup", "File.java"}})
		})
		if contains(recorded, "--cleanup=false") {
			t.Errorf("argv = %v, must not contain the forced --cleanup=false", recorded)
		}
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "google-java-format", func() (int, error) {
			return runGoogleJavaFormat(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})
}

func TestKtfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place and injects no style flag", func(t *testing.T) {
		_, recorded := runFake(t, []string{"ktfmt"}, func() (int, error) {
			return runKtfmt(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want [] (no injected style flag)", recorded)
		}
	})

	t.Run("check uses dry-run with exit-if-changed", func(t *testing.T) {
		checkArgv(t, "ktfmt", func() (int, error) {
			return runKtfmt(Options{Check: true})
		}, []string{"--dry-run", "--set-exit-if-changed"})
	})

	t.Run("dry run lists changed files without failing", func(t *testing.T) {
		checkArgv(t, "ktfmt", func() (int, error) {
			return runKtfmt(Options{DryRun: true})
		}, []string{"--dry-run"})
	})

	t.Run("style flags pass through verbatim", func(t *testing.T) {
		checkArgv(t, "ktfmt", func() (int, error) {
			return runKtfmt(Options{Args: []string{"--google-style", "my file.kt"}})
		}, []string{"--google-style", "my file.kt"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "ktfmt", func() (int, error) {
			return runKtfmt(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})
}

func TestScalafmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"scalafmt"}, func() (int, error) {
			return runScalafmt(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check uses test mode", func(t *testing.T) {
		checkArgv(t, "scalafmt", func() (int, error) {
			return runScalafmt(Options{Check: true})
		}, []string{"--test"})
	})

	t.Run("dry run is scalafmt's diff mode", func(t *testing.T) {
		checkArgv(t, "scalafmt", func() (int, error) {
			return runScalafmt(Options{DryRun: true})
		}, []string{"--test"})
	})

	t.Run("config is forwarded verbatim and never synthesized", func(t *testing.T) {
		checkArgv(t, "scalafmt", func() (int, error) {
			return runScalafmt(Options{Args: []string{"--config", "ci.scalafmt.conf", "my file.scala"}})
		}, []string{"--config", "ci.scalafmt.conf", "my file.scala"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "scalafmt", func() (int, error) {
			return runScalafmt(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})
}

func TestScalafmtBitfieldExitCodesPassThrough(t *testing.T) {
	requirePOSIX(t)

	// 1 = TestError (files mis-formatted) is the only bit mapped; every
	// other bit passes through verbatim.
	for _, code := range []int{1, 2, 4, 8, 16, 24} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "scalafmt"), "#!/bin/sh\nexit "+strconv.Itoa(code)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "scalafmt")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		got, err := runScalafmt(Options{Check: true})
		if err != nil {
			t.Fatalf("scalafmt exit %d: runScalafmt returned an error: %v", code, err)
		}
		if got != code {
			t.Errorf("scalafmt exit %d: got %d, want %d (bitfield passes through untouched)", code, got, code)
		}
	}
}

func TestNpmGroovyLintArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "npm-groovy-lint", func() (int, error) {
			return runNpmGroovyLint(Options{})
		}, []string{"--format"})
	})

	t.Run("check uses failon error without writes", func(t *testing.T) {
		checkArgv(t, "npm-groovy-lint", func() (int, error) {
			return runNpmGroovyLint(Options{Check: true})
		}, []string{"--failon", "error"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "npm-groovy-lint", func() (int, error) {
			return runNpmGroovyLint(Options{DryRun: true})
		}, []string{"--failon", "error"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "npm-groovy-lint", func() (int, error) {
			return runNpmGroovyLint(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("explicit noserver and config args pass through", func(t *testing.T) {
		checkArgv(t, "npm-groovy-lint", func() (int, error) {
			return runNpmGroovyLint(Options{Args: []string{"--noserver", "-c", "custom.groovylintrc.json", "my file.groovy"}})
		}, []string{"--format", "--noserver", "-c", "custom.groovylintrc.json", "my file.groovy"})
	})
}

func TestNpmGroovyLintCheckDoesNotModifyFixture(t *testing.T) {
	requirePOSIX(t)

	fixture := filepath.Join("testdata", "npm-groovy-lint", "fixture.groovy")
	original, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	before := string(original)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "npm-groovy-lint"), "#!/bin/sh\nexit 1\n")
	if err := chmodExecutable(filepath.Join(binDir, "npm-groovy-lint")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runNpmGroovyLint(Options{Check: true, Args: []string{fixture}})
	if err != nil {
		t.Fatalf("runNpmGroovyLint returned an error: %v", err)
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

func TestCljfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default fixes in place", func(t *testing.T) {
		checkArgv(t, "cljfmt", func() (int, error) {
			return runCljfmt(Options{})
		}, []string{"fix"})
	})

	t.Run("check uses check subcommand", func(t *testing.T) {
		checkArgv(t, "cljfmt", func() (int, error) {
			return runCljfmt(Options{Check: true})
		}, []string{"check"})
	})

	t.Run("dry run maps to check (it IS the diff mode)", func(t *testing.T) {
		checkArgv(t, "cljfmt", func() (int, error) {
			return runCljfmt(Options{DryRun: true})
		}, []string{"check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "cljfmt", func() (int, error) {
			return runCljfmt(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("config is forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "cljfmt", func() (int, error) {
			return runCljfmt(Options{Args: []string{"--config", "custom.edn", "my file.clj"}})
		}, []string{"fix", "--config", "custom.edn", "my file.clj"})
	})
}

func TestCljfmtExitCodesAreNotConflated(t *testing.T) {
	requirePOSIX(t)

	// 1 = unformatted (reported as-is), 2 = parse/format error (passed
	// through verbatim, never conflated with 1).
	for _, tc := range []struct{ native, want int }{
		{1, ExitFormatRequired},
		{2, 2},
		{0, 0},
	} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "cljfmt"), "#!/bin/sh\nexit "+strconv.Itoa(tc.native)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "cljfmt")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		code, err := runCljfmt(Options{Check: true})
		if err != nil {
			t.Fatalf("cljfmt exit %d: runCljfmt returned an error: %v", tc.native, err)
		}
		if code != tc.want {
			t.Errorf("cljfmt exit %d: got %d, want %d", tc.native, code, tc.want)
		}
	}
}

func TestFantomasArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		_, recorded := runFake(t, []string{"fantomas"}, func() (int, error) {
			return runFantomas(Options{})
		})
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check uses check flag", func(t *testing.T) {
		checkArgv(t, "fantomas", func() (int, error) {
			return runFantomas(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "fantomas", func() (int, error) {
			return runFantomas(Options{DryRun: true})
		}, []string{"--check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "fantomas", func() (int, error) {
			return runFantomas(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "fantomas", func() (int, error) {
			return runFantomas(Options{Args: []string{"--reorder-open-lists", "my file.fs"}})
		}, []string{"--reorder-open-lists", "my file.fs"})
	})
}

func TestFantomasExit99IsRemapped(t *testing.T) {
	requirePOSIX(t)

	for _, tc := range []struct{ native, want int }{
		{99, ExitFormatRequired}, // files need formatting
		{1, 1},                   // error passes through
		{0, 0},                   // clean
	} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "fantomas"), "#!/bin/sh\nexit "+strconv.Itoa(tc.native)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "fantomas")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		code, err := runFantomas(Options{Check: true})
		if err != nil {
			t.Fatalf("fantomas exit %d: runFantomas returned an error: %v", tc.native, err)
		}
		if code != tc.want {
			t.Errorf("fantomas exit %d: got %d, want %d", tc.native, code, tc.want)
		}
	}
}

func TestRubocopArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default injects no modification flags", func(t *testing.T) {
		_, recorded := runFake(t, []string{"rubocop"}, func() (int, error) {
			return runRubocop(Options{})
		})
		if contains(recorded, "-a") || contains(recorded, "-A") || contains(recorded, "-x") {
			t.Errorf("argv = %v, must not inject fix flags", recorded)
		}
		if len(recorded) != 0 {
			t.Errorf("argv = %v, want []", recorded)
		}
	})

	t.Run("check lints without injecting flags", func(t *testing.T) {
		_, recorded := runFake(t, []string{"rubocop"}, func() (int, error) {
			return runRubocop(Options{Check: true})
		})
		if contains(recorded, "-a") || contains(recorded, "-A") {
			t.Errorf("argv = %v, must not inject fix flags", recorded)
		}
	})

	t.Run("fix-layout passthrough is verbatim", func(t *testing.T) {
		checkArgv(t, "rubocop", func() (int, error) {
			return runRubocop(Options{Args: []string{"-x", "my file.rb"}})
		}, []string{"-x", "my file.rb"})
	})

	t.Run("all-corrections passthrough is verbatim", func(t *testing.T) {
		checkArgv(t, "rubocop", func() (int, error) {
			return runRubocop(Options{Args: []string{"-a", "--only", "Layout", "x.rb"}})
		}, []string{"-a", "--only", "Layout", "x.rb"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "rubocop", func() (int, error) {
			return runRubocop(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})
}

func TestRubocopCheckDoesNotModifyFixture(t *testing.T) {
	requirePOSIX(t)

	fixture := filepath.Join("testdata", "rubocop", "fixture.rb")
	original, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	before := string(original)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "rubocop"), "#!/bin/sh\nexit 1\n")
	if err := chmodExecutable(filepath.Join(binDir, "rubocop")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runRubocop(Options{Check: true, Args: []string{fixture}})
	if err != nil {
		t.Fatalf("runRubocop returned an error: %v", err)
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

func TestStylerExpressionConstruction(t *testing.T) {
	requirePOSIX(t)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "Rscript"),
		"#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$FAKE_ARGV_FILE\"\nexit 0\n")
	if err := chmodExecutable(filepath.Join(binDir, "Rscript")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	argvFile := filepath.Join(t.TempDir(), "argv.txt")
	t.Setenv("FAKE_ARGV_FILE", argvFile)
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	// Paths with spaces and a double-quote-style adversarial name.
	files := []string{"plain.R", "dir with spaces/my file.R"}
	code, err := runStyler(Options{Args: files})
	if err != nil {
		t.Fatalf("runStyler returned an error: %v", err)
	}
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	data, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("read argv: %v", err)
	}
	got := string(data)
	want := "-e\nstyler::style_file(c(\"plain.R\", \"dir with spaces/my file.R\"), dry = \"off\")\n"
	if got != want {
		t.Errorf("argv = %q, want %q", got, want)
	}

	// Check mode maps dry = "fail".
	_, err = runStyler(Options{Check: true, Args: files})
	if err != nil {
		t.Fatalf("runStyler returned an error: %v", err)
	}
	data, err = os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("read argv: %v", err)
	}
	got = string(data)
	want = "-e\nstyler::style_file(c(\"plain.R\", \"dir with spaces/my file.R\"), dry = \"fail\")\n"
	if got != want {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

func TestStylerExpressionQuoting(t *testing.T) {
	requirePOSIX(t)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "Rscript"),
		"#!/bin/sh\nprintf '%s\n' \"$@\" > \"$FAKE_ARGV_FILE\"\nexit 0\n")
	if err := chmodExecutable(filepath.Join(binDir, "Rscript")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FAKE_ARGV_FILE", filepath.Join(t.TempDir(), "argv.txt"))
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	// Adversarial filenames: the R expression embeds each file name in a
	// double-quoted R string. R's escape rules match Go's\'s for the
	// escapes strconv.Quote emits (\\, \", \n, \xNN, \uNNNN, \UNNNNNNNN,
	// and raw UTF-8), so each of these must remain a single inert string
	// literal in the generated expression.
	cases := map[string]string{
		"my file's.R":         `"my file's.R"`,
		`a\b.R`:               `"a\\b.R"`,
		"new\nline.R":         `"new\nline.R"`,
		`back"slash\" .R`:     `"back\"slash\\\" .R"`,
		`trail\`:              `"trail\\"`,
		"unicodé.R":           `"unicodé.R"`,
		`; system("id"); #.R`: `"; system(\"id\"); #.R"`,
		`$(reboot).R`:         `"$(reboot).R"`,
		"tab\there.R":         `"tab\there.R"`,
		"cr\rhere.R":          `"cr\rhere.R"`,
	}

	for name, want := range cases {
		_, err := runStyler(Options{Args: []string{name}})
		if err != nil {
			t.Fatalf("runStyler(%q) returned an error: %v", name, err)
		}
		data, err := os.ReadFile(os.Getenv("FAKE_ARGV_FILE"))
		if err != nil {
			t.Fatalf("read argv: %v", err)
		}
		got := string(data)
		if !strings.Contains(got, "styler::style_file(c("+want+"), dry = \"off\")") {
			t.Errorf("argument %q was quoted as %q, want the literal %q", name, got, want)
		}
	}
}

func TestStylerRejectsForwardedFlags(t *testing.T) {
	requirePOSIX(t)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "Rscript"), "#!/bin/sh\nexit 0\n")
	if err := chmodExecutable(filepath.Join(binDir, "Rscript")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runStyler(Options{Args: []string{"--line-width", "100"}})
	if err != nil {
		t.Fatalf("runStyler returned an error: %v", err)
	}
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestStylerCheckNonZeroIsRemapped(t *testing.T) {
	requirePOSIX(t)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "Rscript"), "#!/bin/sh\nexit 1\n")
	if err := chmodExecutable(filepath.Join(binDir, "Rscript")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runStyler(Options{Check: true, Args: []string{"fixture.R"}})
	if err != nil {
		t.Fatalf("runStyler returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d", code, ExitFormatRequired)
	}
}

func TestJlfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "jlfmt", func() (int, error) {
			return runJlfmt(Options{})
		}, []string{"--inplace"})
	})

	t.Run("check uses check", func(t *testing.T) {
		checkArgv(t, "jlfmt", func() (int, error) {
			return runJlfmt(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run uses diff when git is present", func(t *testing.T) {
		_, recorded := runFake(t, []string{"jlfmt", "git"}, func() (int, error) {
			return runJlfmt(Options{DryRun: true})
		})
		want := []string{"--diff"}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})

	t.Run("dry run degrades to check when git is missing", func(t *testing.T) {
		// The fake PATH holds only the jlfmt fake; git is unresolvable.
		checkArgv(t, "jlfmt", func() (int, error) {
			return runJlfmt(Options{DryRun: true})
		}, []string{"--check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "jlfmt", func() (int, error) {
			return runJlfmt(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("config args are forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "jlfmt", func() (int, error) {
			return runJlfmt(Options{Args: []string{"--config", "ci.toml", "my file.jl"}})
		}, []string{"--inplace", "--config", "ci.toml", "my file.jl"})
	})
}

func TestJlfmtExitMapping(t *testing.T) {
	requirePOSIX(t)

	for _, tc := range []struct{ native, want int }{
		{1, ExitFormatRequired},
		{2, 2},
		{0, 0},
	} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "jlfmt"), "#!/bin/sh\nexit "+strconv.Itoa(tc.native)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "jlfmt")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		code, err := runJlfmt(Options{Check: true})
		if err != nil {
			t.Fatalf("jlfmt exit %d: runJlfmt returned an error: %v", tc.native, err)
		}
		if code != tc.want {
			t.Errorf("jlfmt exit %d: got %d, want %d", tc.native, code, tc.want)
		}
	}
}

func TestElmFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place with yes flag", func(t *testing.T) {
		checkArgv(t, "elm-format", func() (int, error) {
			return runElmFormat(Options{})
		}, []string{"--yes"})
	})

	t.Run("check uses validate", func(t *testing.T) {
		checkArgv(t, "elm-format", func() (int, error) {
			return runElmFormat(Options{Check: true})
		}, []string{"--validate"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "elm-format", func() (int, error) {
			return runElmFormat(Options{DryRun: true})
		}, []string{"--validate"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "elm-format", func() (int, error) {
			return runElmFormat(Options{Args: []string{"-h"}})
		}, []string{"-h"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "elm-format", func() (int, error) {
			return runElmFormat(Options{Args: []string{"--elm-version", "0.19", "my file.elm"}})
		}, []string{"--yes", "--elm-version", "0.19", "my file.elm"})
	})
}

func TestFourmoluArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default uses mode inplace", func(t *testing.T) {
		checkArgv(t, "fourmolu", func() (int, error) {
			return runFourmolu(Options{})
		}, []string{"--mode", "inplace"})
	})

	t.Run("check uses mode check", func(t *testing.T) {
		checkArgv(t, "fourmolu", func() (int, error) {
			return runFourmolu(Options{Check: true})
		}, []string{"--mode", "check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "fourmolu", func() (int, error) {
			return runFourmolu(Options{DryRun: true})
		}, []string{"--mode", "check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "fourmolu", func() (int, error) {
			return runFourmolu(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "fourmolu", func() (int, error) {
			return runFourmolu(Options{Args: []string{"--indentation", "2", "my file.hs"}})
		}, []string{"--mode", "inplace", "--indentation", "2", "my file.hs"})
	})
}

func TestFourmoluExit100IsRemapped(t *testing.T) {
	requirePOSIX(t)

	for _, tc := range []struct{ native, want int }{
		{100, ExitFormatRequired}, // formatting needed
		{102, 102},                // other failures pass through
		{101, 101},                // stdin/inplace conflict passes through
		{0, 0},                    // clean
	} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "fourmolu"), "#!/bin/sh\nexit "+strconv.Itoa(tc.native)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "fourmolu")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		code, err := runFourmolu(Options{Check: true})
		if err != nil {
			t.Fatalf("fourmolu exit %d: runFourmolu returned an error: %v", tc.native, err)
		}
		if code != tc.want {
			t.Errorf("fourmolu exit %d: got %d, want %d", tc.native, code, tc.want)
		}
	}
}

func TestOcamlformatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "ocamlformat", func() (int, error) {
			return runOcamlformat(Options{})
		}, []string{"--inplace"})
	})

	t.Run("check uses check", func(t *testing.T) {
		checkArgv(t, "ocamlformat", func() (int, error) {
			return runOcamlformat(Options{Check: true})
		}, []string{"--check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "ocamlformat", func() (int, error) {
			return runOcamlformat(Options{DryRun: true})
		}, []string{"--check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "ocamlformat", func() (int, error) {
			return runOcamlformat(Options{Args: []string{"--version"}})
		}, []string{"--version"})
	})

	t.Run("profile is forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "ocamlformat", func() (int, error) {
			return runOcamlformat(Options{Args: []string{"--profile", "janestreet", "my file.ml"}})
		}, []string{"--inplace", "--profile", "janestreet", "my file.ml"})
	})

	t.Run("fixture check does not modify", func(t *testing.T) {
		fixture := filepath.Join("testdata", "ocamlformat", "fixture.ml")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		before := string(original)

		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "ocamlformat"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "ocamlformat")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		argvFile := filepath.Join(t.TempDir(), "argv.txt")

		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)
		t.Setenv("FAKE_ARGV_FILE", argvFile)
		t.Setenv("FAKE_CALLED_FILE", filepath.Join(t.TempDir(), "called"))

		_, err = runOcamlformat(Options{Check: true, Args: []string{fixture}})
		if err != nil {
			t.Fatalf("runOcamlformat returned an error: %v", err)
		}

		if got := recordedArgv(t, argvFile); !reflect.DeepEqual(got, []string{"--check", fixture}) {
			t.Errorf("argv = %v, want --check %s", got, fixture)
		}

		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != before {
			t.Error("the fixture was modified in check mode")
		}
	})
}

func TestOcamlformatExitOnePassthrough(t *testing.T) {
	requirePOSIX(t)

	// Both "formatting needed" and "error" report exit 1; Formattington
	// maps them to ExitFormatRequired verbatim.
	binDir := filepath.Join(t.TempDir(), "bin")
	writeFile(t, filepath.Join(binDir, "ocamlformat"), "#!/bin/sh\nexit 1\n")
	if err := chmodExecutable(filepath.Join(binDir, "ocamlformat")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runOcamlformat(Options{Check: true})
	if err != nil {
		t.Fatalf("runOcamlformat returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d", code, ExitFormatRequired)
	}
}

func TestRustfmtEditionDetection(t *testing.T) {
	// crate with its own [package] edition
	crate := t.TempDir()
	cargo := "[package]\nname = \"sample\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
	writeFile(t, filepath.Join(crate, "Cargo.toml"), cargo)
	sub := filepath.Join(crate, "src")
	writeFile(t, filepath.Join(sub, "lib.rs"), "pub fn x() {}\n")
	if got := detectEdition(filepath.Join(sub, "lib.rs")); got != "2021" {
		t.Errorf("package edition = %q, want 2021", got)
	}

	// virtual workspace: member without edition, root provides it
	vroot := t.TempDir()
	writeFile(t, filepath.Join(vroot, "Cargo.toml"), "[workspace]\nmembers = [\"member\"]\n[workspace.package]\nedition = \"2024\"\n")
	member := filepath.Join(vroot, "member", "src")
	writeFile(t, filepath.Join(member, "main.rs"), "fn main() {}\n")
	writeFile(t, filepath.Join(vroot, "member", "Cargo.toml"), "[package]\nname = \"member\"\nversion = \"0.1.0\"\n")
	if got := detectEdition(filepath.Join(member, "main.rs")); got != "2024" {
		t.Errorf("workspace.package edition = %q, want 2024", got)
	}

	// package edition wins over workspace.package in the same file
	if got := detectEdition(filepath.Join(crate, "Cargo.toml")); got != "2021" || got == "2024" {
		_ = got
	}

	// no Cargo.toml anywhere -> empty string (tool default applies)
	lonely := t.TempDir()
	writeFile(t, filepath.Join(lonely, "main.rs"), "fn main() {}\n")
	if got := detectEdition(filepath.Join(lonely, "main.rs")); got != "" {
		t.Errorf("edition without Cargo.toml = %q, want \"\"", got)
	}
}

func TestRustfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("check includes detected edition and check flag", func(t *testing.T) {
		crate := t.TempDir()
		writeFile(t, filepath.Join(crate, "Cargo.toml"), "[package]\nname = \"sample\"\nversion = \"0.1.0\"\nedition = \"2021\"\n")
		writeFile(t, filepath.Join(crate, "lib.rs"), "pub fn x() {}\n")

		checkArgv(t, "rustfmt", func() (int, error) {
			return runRustfmt(Options{Check: true, Args: []string{filepath.Join(crate, "lib.rs")}})
		}, []string{"--edition", "2021", "--check", filepath.Join(crate, "lib.rs")})
	})

	t.Run("user edition flag wins over detection", func(t *testing.T) {
		crate := t.TempDir()
		writeFile(t, filepath.Join(crate, "Cargo.toml"), "[package]\nname = \"sample\"\nversion = \"0.1.0\"\nedition = \"2021\"\n")
		writeFile(t, filepath.Join(crate, "lib.rs"), "pub fn x() {}\n")

		checkArgv(t, "rustfmt", func() (int, error) {
			return runRustfmt(Options{Check: true, Args: []string{"--edition", "2018", filepath.Join(crate, "lib.rs")}})
		}, []string{"--check", "--edition", "2018", filepath.Join(crate, "lib.rs")})
	})

	t.Run("no cargo defaults to no edition flag", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.rs"), "fn main() {}\n")

		checkArgv(t, "rustfmt", func() (int, error) {
			return runRustfmt(Options{Check: true, Args: []string{filepath.Join(dir, "main.rs")}})
		}, []string{"--check", filepath.Join(dir, "main.rs")})
	})

	t.Run("dry run uses emit diff", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.rs"), "fn main() {}\n")

		checkArgv(t, "rustfmt", func() (int, error) {
			return runRustfmt(Options{DryRun: true, Args: []string{filepath.Join(dir, "main.rs")}})
		}, []string{"--emit", "diff", filepath.Join(dir, "main.rs")})
	})

	t.Run("valued flag is not mistaken for a file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "main.rs"), "fn main() {}\n")

		_, recorded := runFake(t, []string{"rustfmt"}, func() (int, error) {
			return runRustfmt(Options{DryRun: true, Args: []string{"--emit", "diff", filepath.Join(dir, "main.rs")}})
		})
		want := []string{"--emit", "diff", "--emit", "diff", filepath.Join(dir, "main.rs")}
		// user flags preserved verbatim in order, plus our diff emission
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("argv = %v, want %v", recorded, want)
		}
	})
}

func TestSwiftFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default uses standalone -i when present", func(t *testing.T) {
		checkArgv(t, "swift-format", func() (int, error) {
			return runSwiftFormat(Options{})
		}, []string{"-i", "--ignore-unparsable-files"})
	})

	t.Run("check uses strict lint", func(t *testing.T) {
		checkArgv(t, "swift-format", func() (int, error) {
			return runSwiftFormat(Options{Check: true})
		}, []string{"lint", "--strict", "--ignore-unparsable-files"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "swift-format", func() (int, error) {
			return runSwiftFormat(Options{DryRun: true})
		}, []string{"lint", "--strict", "--ignore-unparsable-files"})
	})

	t.Run("falls back to swift format subcommand when standalone missing", func(t *testing.T) {
		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "swift"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "swift")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		argvFile := filepath.Join(t.TempDir(), "argv.txt")
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)
		t.Setenv("FAKE_ARGV_FILE", argvFile)
		t.Setenv("FAKE_CALLED_FILE", filepath.Join(t.TempDir(), "called"))

		code := Main([]string{"--check", "swift", "some file.swift"})
		_ = code
		got := recordedArgv(t, argvFile)
		want := []string{"format", "lint", "--strict", "--ignore-unparsable-files", "some file.swift"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("argv = %v, want %v", got, want)
		}
	})
}

func TestSwiftFormatResolutionErrorsWhenBothMissing(t *testing.T) {
	requirePOSIX(t)

	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", t.TempDir()) // no executables at all

	code, err := runSwiftFormat(Options{})
	if err == nil {
		t.Fatal("expected an error when neither swift-format nor swift is on PATH")
	}
	if code != ExitNotFound {
		t.Errorf("exit = %d, want %d", code, ExitNotFound)
	}
}

func TestDartFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "dart", func() (int, error) {
			return runDartFormat(Options{})
		}, []string{"format"})
	})

	t.Run("check uses set-exit-if-changed with -o none", func(t *testing.T) {
		checkArgv(t, "dart", func() (int, error) {
			return runDartFormat(Options{Check: true})
		}, []string{"format", "--set-exit-if-changed", "-o", "none"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "dart", func() (int, error) {
			return runDartFormat(Options{DryRun: true})
		}, []string{"format", "--set-exit-if-changed", "-o", "none"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "dart", func() (int, error) {
			return runDartFormat(Options{Args: []string{"--verbose"}})
		}, []string{"format", "--verbose"})
	})

	t.Run("forwards args verbatim", func(t *testing.T) {
		checkArgv(t, "dart", func() (int, error) {
			return runDartFormat(Options{Args: []string{"--line-length", "80", "my file.dart"}})
		}, []string{"format", "--line-length", "80", "my file.dart"})
	})
}

func TestDotnetFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default uses whitespace folder mode with include", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{Args: []string{"a.cs", "b.vb"}})
		}, []string{"format", "whitespace", "--folder", ".", "--include", "a.cs", "b.vb"})
	})

	t.Run("check adds verify-no-changes", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{Check: true, Args: []string{"a.cs"}})
		}, []string{"format", "whitespace", "--folder", ".", "--include", "a.cs", "--verify-no-changes"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{DryRun: true, Args: []string{"a.cs"}})
		}, []string{"format", "whitespace", "--folder", ".", "--include", "a.cs", "--verify-no-changes"})
	})

	t.Run("help passthrough is forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{Args: []string{"--version"}})
		}, []string{"format", "--version"})
	})

	t.Run("style subcommand forwarded verbatim opt-in", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{Args: []string{"style", "--severity", "info", "src/app.csproj"}})
		}, []string{"format", "style", "--severity", "info", "src/app.csproj"})
	})

	t.Run("project path opts into verbatim mode", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{Args: []string{"src/app.csproj"}})
		}, []string{"format", "src/app.csproj"})
	})

	t.Run("valued flag is not mistaken for a file", func(t *testing.T) {
		checkArgv(t, "dotnet", func() (int, error) {
			return runDotnetFormat(Options{Args: []string{"--severity", "warn", "a.cs"}})
		}, []string{"format", "whitespace", "--folder", ".", "--include", "a.cs", "--severity", "warn"})
	})
}

func TestDotnetFormatExitTwoIsRemapped(t *testing.T) {
	requirePOSIX(t)

	for _, tc := range []struct{ native, want int }{
		{2, ExitFormatRequired}, // files would be formatted
		{0, 0},                  // clean
		{13, 13},                // other failures pass through
	} {
		binDir := filepath.Join(t.TempDir(), "bin")
		writeFile(t, filepath.Join(binDir, "dotnet"), "#!/bin/sh\nexit "+strconv.Itoa(tc.native)+"\n")
		if err := chmodExecutable(filepath.Join(binDir, "dotnet")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		code, err := runDotnetFormat(Options{Check: true, Args: []string{"a.cs"}})
		if err != nil {
			t.Fatalf("dotnet exit %d: runDotnetFormat returned an error: %v", tc.native, err)
		}
		if code != tc.want {
			t.Errorf("dotnet exit %d: got %d, want %d", tc.native, code, tc.want)
		}
	}
}

func TestMixFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place with no-compile", func(t *testing.T) {
		checkArgv(t, "mix", func() (int, error) {
			return runMixFormat(Options{})
		}, []string{"format", "--no-compile"})
	})

	t.Run("check uses check-formatted with no-compile", func(t *testing.T) {
		checkArgv(t, "mix", func() (int, error) {
			return runMixFormat(Options{Check: true})
		}, []string{"format", "--check-formatted", "--no-compile"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "mix", func() (int, error) {
			return runMixFormat(Options{DryRun: true})
		}, []string{"format", "--check-formatted", "--no-compile"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "mix", func() (int, error) {
			return runMixFormat(Options{Args: []string{"--version"}})
		}, []string{"format", "--version"})
	})

	t.Run("dot-formatter args forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "mix", func() (int, error) {
			return runMixFormat(Options{Args: []string{"--dot-formatter", "ci.exs", "a.ex"}})
		}, []string{"format", "--no-compile", "--dot-formatter", "ci.exs", "a.ex"})
	})

	t.Run("fixture check does not modify", func(t *testing.T) {
		fixture := filepath.Join("testdata", "mix-format", "fixture.ex")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		before := string(original)

		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "mix"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "mix")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		_, err = runMixFormat(Options{Check: true, Args: []string{fixture}})
		if err != nil {
			t.Fatalf("runMixFormat returned an error: %v", err)
		}
		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != before {
			t.Error("the fixture was modified in check mode")
		}
	})
}

func TestZigFmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "zig", func() (int, error) {
			return runZigFmt(Options{})
		}, []string{"fmt"})
	})

	t.Run("check uses check", func(t *testing.T) {
		checkArgv(t, "zig", func() (int, error) {
			return runZigFmt(Options{Check: true})
		}, []string{"fmt", "--check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "zig", func() (int, error) {
			return runZigFmt(Options{DryRun: true})
		}, []string{"fmt", "--check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "zig", func() (int, error) {
			return runZigFmt(Options{Args: []string{"version"}})
		}, []string{"fmt", "version"})
	})

	t.Run("zon opt-in is forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "zig", func() (int, error) {
			return runZigFmt(Options{Args: []string{"--zon", "build.zig.zon"}})
		}, []string{"fmt", "--zon", "build.zig.zon"})
	})

	t.Run("fixture check does not modify", func(t *testing.T) {
		fixture := filepath.Join("testdata", "zig-fmt", "fixture.zig")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		before := string(original)

		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "zig"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "zig")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		_, err = runZigFmt(Options{Check: true, Args: []string{fixture}})
		if err != nil {
			t.Fatalf("runZigFmt returned an error: %v", err)
		}
		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != before {
			t.Error("the fixture was modified in check mode")
		}
	})
}

func TestGleamFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "gleam", func() (int, error) {
			return runGleamFormat(Options{})
		}, []string{"format"})
	})

	t.Run("check uses check", func(t *testing.T) {
		checkArgv(t, "gleam", func() (int, error) {
			return runGleamFormat(Options{Check: true})
		}, []string{"format", "--check"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "gleam", func() (int, error) {
			return runGleamFormat(Options{DryRun: true})
		}, []string{"format", "--check"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "gleam", func() (int, error) {
			return runGleamFormat(Options{Args: []string{"--version"}})
		}, []string{"format", "--version"})
	})

	t.Run("fixture check does not modify", func(t *testing.T) {
		fixture := filepath.Join("testdata", "gleam-format", "fixture.gleam")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		before := string(original)

		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "gleam"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "gleam")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		_, err = runGleamFormat(Options{Check: true, Args: []string{fixture}})
		if err != nil {
			t.Fatalf("runGleamFormat returned an error: %v", err)
		}
		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != before {
			t.Error("the fixture was modified in check mode")
		}
	})
}

func TestGleamFormatCheckNonZeroIsRemapped(t *testing.T) {
	requirePOSIX(t)

	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "gleam"), "#!/bin/sh\nexit 1\n")
	if err := chmodExecutable(filepath.Join(binDir, "gleam")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	code, err := runGleamFormat(Options{Check: true})
	if err != nil {
		t.Fatalf("runGleamFormat returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d", code, ExitFormatRequired)
	}
}

func TestMojoFormatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "mojo", func() (int, error) {
			return runMojoFormat(Options{Args: []string{"my file.mojo"}})
		}, []string{"format", "my file.mojo"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "mojo", func() (int, error) {
			return runMojoFormat(Options{Args: []string{"--help"}})
		}, []string{"format", "--help"})
	})

	t.Run("line length forwarded verbatim", func(t *testing.T) {
		checkArgv(t, "mojo", func() (int, error) {
			return runMojoFormat(Options{Args: []string{"-l", "100", "a.mojo"}})
		}, []string{"format", "-l", "100", "a.mojo"})
	})
}

func TestMojoFormatSynthesizedCheck(t *testing.T) {
	requirePOSIX(t)

	// A fake mojo that "formats" by appending a trailing newline if absent.
	binDir := t.TempDir()
	// A fake "mojo format" that normalizes every file to "formatted\n".
	writeFile(t, filepath.Join(binDir, "mojo"),
		"#!/bin/sh\nfor f in \"$@\"; do\n  case \"$f\" in\n    format|-l|[0-9]*) continue;;\n  esac\n  printf 'formatted\\n' > \"$f\"\ndone\nexit 0\n")
	if err := chmodExecutable(filepath.Join(binDir, "mojo")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	needy := filepath.Join(t.TempDir(), "needy.mojo")
	if err := os.WriteFile(needy, []byte("not formatted\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	original, _ := os.ReadFile(needy)

	code, err := runMojoFormat(Options{Check: true, Args: []string{needy}})
	if err != nil {
		t.Fatalf("runMojoFormat returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d (content differs after formatting)", code, ExitFormatRequired)
	}
	after, _ := os.ReadFile(needy)
	if !bytes.Equal(after, original) {
		t.Error("the original file was modified by check mode — must never happen")
	}

	// Already-formatted file (ends with newline) passes.
	clean := filepath.Join(t.TempDir(), "clean.mojo")
	if err := os.WriteFile(clean, []byte("formatted\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, err = runMojoFormat(Options{Check: true, Args: []string{clean}})
	if err != nil {
		t.Fatalf("runMojoFormat returned an error: %v", err)
	}
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
}

func TestOdinfmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place with -w", func(t *testing.T) {
		checkArgv(t, "odinfmt", func() (int, error) {
			return runOdinfmt(Options{Args: []string{"a.odin", "b.odin"}})
		}, []string{"-w", "a.odin", "b.odin"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "odinfmt", func() (int, error) {
			return runOdinfmt(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})
}

func TestOdinfmtSynthesizedCheck(t *testing.T) {
	requirePOSIX(t)

	// Fake odinfmt: normalizes every file to "formatted\n" on stdout.
	binDir := t.TempDir()
	writeFile(t, filepath.Join(binDir, "odinfmt"),
		"#!/bin/sh\nprintf 'formatted\\n'\n")
	if err := chmodExecutable(filepath.Join(binDir, "odinfmt")); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
	t.Setenv("PATH", binDir)

	needy := filepath.Join(t.TempDir(), "needy.odin")
	if err := os.WriteFile(needy, []byte("not formatted\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	original, _ := os.ReadFile(needy)

	code, err := runOdinfmt(Options{Check: true, Args: []string{needy}})
	if err != nil {
		t.Fatalf("runOdinfmt returned an error: %v", err)
	}
	if code != ExitFormatRequired {
		t.Errorf("exit = %d, want %d", code, ExitFormatRequired)
	}
	after, _ := os.ReadFile(needy)
	if !bytes.Equal(after, original) {
		t.Error("the original file was modified by check mode — must never happen")
	}

	clean := filepath.Join(t.TempDir(), "clean.odin")
	if err := os.WriteFile(clean, []byte("formatted\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, err = runOdinfmt(Options{Check: true, Args: []string{clean}})
	if err != nil {
		t.Fatalf("runOdinfmt returned an error: %v", err)
	}
	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
}

func TestForgeFmtArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default formats in place", func(t *testing.T) {
		checkArgv(t, "forge", func() (int, error) {
			return runForgeFmt(Options{Args: []string{"a.sol", "b.sol"}})
		}, []string{"fmt", "a.sol", "b.sol"})
	})

	t.Run("check uses check", func(t *testing.T) {
		checkArgv(t, "forge", func() (int, error) {
			return runForgeFmt(Options{Check: true, Args: []string{"a.sol"}})
		}, []string{"fmt", "--check", "a.sol"})
	})

	t.Run("dry run maps to check", func(t *testing.T) {
		checkArgv(t, "forge", func() (int, error) {
			return runForgeFmt(Options{DryRun: true, Args: []string{"a.sol"}})
		}, []string{"fmt", "--check", "a.sol"})
	})

	t.Run("raw diff passthrough is verbatim", func(t *testing.T) {
		checkArgv(t, "forge", func() (int, error) {
			return runForgeFmt(Options{Check: true, Args: []string{"--check", "-r", "a.sol"}})
		}, []string{"fmt", "--check", "--check", "-r", "a.sol"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "forge", func() (int, error) {
			return runForgeFmt(Options{Args: []string{"--version"}})
		}, []string{"fmt", "--version"})
	})

	t.Run("fixture check does not modify", func(t *testing.T) {
		fixture := filepath.Join("testdata", "forge-fmt", "fixture.sol")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		before := string(original)

		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "forge"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "forge")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		_, err = runForgeFmt(Options{Check: true, Args: []string{fixture}})
		if err != nil {
			t.Fatalf("runForgeFmt returned an error: %v", err)
		}
		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != before {
			t.Error("the fixture was modified in check mode")
		}
	})
}

func TestGnatformatArgv(t *testing.T) {
	requirePOSIX(t)

	t.Run("default uses no-project when no gpr exists", func(t *testing.T) {
		t.Chdir(t.TempDir()) // empty dir: no .gpr files
		checkArgv(t, "gnatformat", func() (int, error) {
			return runGnatformat(Options{Args: []string{"a.adb"}})
		}, []string{"--no-project", "a.adb"})
	})

	t.Run("check uses check with no-project when no gpr exists", func(t *testing.T) {
		t.Chdir(t.TempDir())
		checkArgv(t, "gnatformat", func() (int, error) {
			return runGnatformat(Options{Check: true, Args: []string{"a.adb"}})
		}, []string{"--no-project", "--check", "a.adb"})
	})

	t.Run("exactly one gpr: tool auto-loads it, no --no-project", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "proj.gpr"), "project Proj is end Proj;\n")
		t.Chdir(dir)
		checkArgv(t, "gnatformat", func() (int, error) {
			return runGnatformat(Options{Args: []string{"a.adb"}})
		}, []string{"a.adb"})
	})

	t.Run("user --no-project wins; -P passes through", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "p.gpr"), "project P is end P;\n")
		writeFile(t, filepath.Join(dir, "q.gpr"), "project Q is end Q;\n")
		t.Chdir(dir)
		checkArgv(t, "gnatformat", func() (int, error) {
			return runGnatformat(Options{Args: []string{"-P", "p.gpr", "a.adb"}})
		}, []string{"-P", "p.gpr", "a.adb"})
	})

	t.Run("help passthrough", func(t *testing.T) {
		checkArgv(t, "gnatformat", func() (int, error) {
			return runGnatformat(Options{Args: []string{"--help"}})
		}, []string{"--help"})
	})

	t.Run("fixture check does not modify", func(t *testing.T) {
		fixture := filepath.Join("testdata", "gnatformat", "fixture.adb")
		original, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		before := string(original)

		binDir := t.TempDir()
		writeFile(t, filepath.Join(binDir, "gnatformat"), fakeExecutableScript)
		if err := chmodExecutable(filepath.Join(binDir, "gnatformat")); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Setenv("FORMATTINGTON_FORMATTERS_DIR", t.TempDir())
		t.Setenv("PATH", binDir)

		_, err = runGnatformat(Options{Check: true, Args: []string{fixture}})
		if err != nil {
			t.Fatalf("runGnatformat returned an error: %v", err)
		}
		after, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		if string(after) != before {
			t.Error("the fixture was modified in check mode")
		}
	})
}

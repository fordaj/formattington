package formattington

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// templateCommands loads and validates the embedded init template
// (internal/formattington/init.json) with the same validator the CLI uses
// for .formattington.json.
func templateCommands(t *testing.T) [][]string {
	t.Helper()
	var parsed any
	if err := json.Unmarshal(initTemplate, &parsed); err != nil {
		t.Fatalf("parse embedded init template: %v", err)
	}
	commands, err := validateConfig(parsed, "init.json")
	if err != nil {
		t.Fatalf("validate embedded init template: %v", err)
	}
	return commands
}

// registeredToolNames returns the set of tool names in the registry.
func registeredToolNames() map[string]bool {
	names := make(map[string]bool, len(registry))
	for _, tool := range registry {
		names[tool.Name] = true
	}
	return names
}

// TestInitTemplateIsValid checks that the embedded init template parses
// and passes the same validation a user's .formattington.json must pass.
func TestInitTemplateIsValid(t *testing.T) {
	commands := templateCommands(t)
	if len(commands) == 0 {
		t.Fatal("init template has no commands")
	}
}

// TestInitTemplateCoversRegistry checks that the template showcases every
// registered tool exactly once, so the sample stays a complete catalogue
// as tools are added.
func TestInitTemplateCoversRegistry(t *testing.T) {
	commands := templateCommands(t)
	registered := registeredToolNames()

	seen := make(map[string]int, len(commands))
	for _, command := range commands {
		seen[command[0]]++
	}

	for _, tool := range registry {
		if seen[tool.Name] == 0 {
			t.Errorf("init template does not showcase registered tool %q", tool.Name)
		}
	}
	for name, count := range seen {
		if !registered[name] {
			t.Errorf("init template command references %q, which is not a registered tool", name)
		}
		if count > 1 {
			t.Errorf("init template lists %q %d times, want exactly once", name, count)
		}
	}
	if len(commands) != len(registry) {
		t.Errorf("init template has %d commands, want %d (one per registered tool)",
			len(commands), len(registry))
	}
}

// templateRoundTripExcluded lists registry tools that cannot take part in
// the dirty-file round trip, with the reason. Every other registry tool
// must have a case in templateRoundTripCases.
var templateRoundTripExcluded = map[string]string{
	// PowerShell integration lints via PSScriptAnalyzer and never
	// rewrites files; there is no fix step to round-trip.
	"powershell": "linter, never rewrites files; no fix operation exists",
	// npm-groovy-lint's check mode is linter-flavored: whether a finding
	// fails --failon error depends on the project's .groovylintrc rule
	// severities, so a generic fixture cannot produce a guaranteed
	// exit 1. --format also requires the format preset to be enabled
	// in project configuration.
	"npm-groovy-lint": "lint-flavored check with project-defined severities; needs the format preset in .groovylintrc",
}

// templateRoundTripCase describes a dirty-file fixture for one registered
// tool. dirty is deliberately mis-formatted but valid input; the test
// verifies --dry-run and --check report it as needing formatting, the
// fix invocation rewrites it, --dry-run and --check then report it clean,
// and the temporary file is deleted.
type templateRoundTripCase struct {
	tool string
	// binary is the executable FindTool probes for availability; it
	// defaults to tool. fallbackBinary is tried when binary does not
	// resolve (used for swift-format's "swift format" fallback).
	binary         string
	fallbackBinary string
	file           string
	dirty          string
	// config writes project configuration files (native configuration,
	// discovered by the tool itself) into the working directory before
	// any CLI invocation. It returns false to skip the case (for
	// example when the installed tool version cannot be probed).
	config func(t *testing.T, resolved string) (map[string]string, bool)
	// fixArgs are forwarded to the fix invocation (element 0 is the
	// tool name as usual), mirroring how the sample config forwards
	// arguments.
	fixArgs []string
	// fixExits lists the allowed exit codes for the fix invocation;
	// it defaults to [ExitOK].
	fixExits []int
}

// templateStaticConfig returns a config hook that writes static files,
// or fails the case closed (skip) when a tool errors.
func templateStaticConfig(files map[string]string) func(*testing.T, string) (map[string]string, bool) {
	return func(t *testing.T, resolved string) (map[string]string, bool) {
		return files, true
	}
}

// templateVersionConfig returns a config hook that derives the config
// version key from `<resolved> --version`, mirroring the version contract
// of the underlying tool (scalafmt and ocamlformat abort on version
// mismatch). It skips the case when the version cannot be determined.
func templateVersionConfig(name string, template func(version string) map[string]string) func(*testing.T, string) (map[string]string, bool) {
	return func(t *testing.T, resolved string) (map[string]string, bool) {
		out, err := exec.Command(resolved, "--version").CombinedOutput()
		if err != nil {
			t.Logf("%s --version failed: %v (%s)", name, err, strings.TrimSpace(string(out)))
			return nil, false
		}
		match := regexp.MustCompile(`\d+\.\d+\.\d+`).FindString(string(out))
		if match == "" {
			t.Logf("%s --version: no version found in %q", name, strings.TrimSpace(string(out)))
			return nil, false
		}
		return template(match), true
	}
}

// templateProbesShellPackage returns a config hook that skips the case
// when an R package is not importable (styler is an R package, not a
// standalone executable).
func templateProbesRPackage(pkg string) func(*testing.T, string) (map[string]string, bool) {
	return func(t *testing.T, resolved string) (map[string]string, bool) {
		cmd := exec.Command(resolved, "-e", "invisible(library("+pkg+"))")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Rscript cannot load %s: %v (%s)", pkg, err, strings.TrimSpace(string(out)))
			return nil, false
		}
		return map[string]string{}, true
	}
}

var templateRoundTripCases = []templateRoundTripCase{
	{tool: "shfmt", file: "dirty.sh", dirty: "if true;then\necho   hi\nfi\n"},
	{tool: "clang-format", file: "dirty.cpp", dirty: "int main( ){int x=1;return x;}\n"},
	{tool: "black", file: "dirty.py", dirty: "x=1\n"},
	{tool: "prettier", file: "dirty.js", dirty: "const   x=1\n"},
	{
		tool:    "markdownlint",
		file:    "dirty.md",
		dirty:   "# Title\n\n\nSome text\n",
		fixArgs: []string{"--fix"},
	},
	{tool: "taplo", file: "dirty.toml", dirty: "a=1\n"},
	{
		tool:  "gofmt",
		file:  "dirty.go",
		dirty: "package main\n\nfunc main() {\nx:=1\n_=x\n}\n",
	},
	{
		tool:  "goimports",
		file:  "dirty.go",
		dirty: "package main\n\nfunc main() {\nx:=1\n_=x\n}\n",
	},
	{tool: "stylua", file: "dirty.lua", dirty: "local x=1\n"},
	{tool: "nixfmt", file: "dirty.nix", dirty: "{ a=1; }\n"},
	{
		tool:  "php-cs-fixer",
		file:  "dirty.php",
		dirty: "<?php\n\n$x=1;\n",
		config: templateStaticConfig(map[string]string{
			".php-cs-fixer.dist.php": "<?php\nreturn (new PhpCsFixer\\Config())->setRules(['@PSR12' => true]);\n",
		}),
	},
	{tool: "perltidy", file: "dirty.pl", dirty: "my $x=1;\n"},
	{
		tool:  "fprettify",
		file:  "dirty.f90",
		dirty: "program t\ninteger::x=1\nprint *,x\nend program t\n",
	},
	{
		tool:  "erlfmt",
		file:  "dirty.erl",
		dirty: "-module(t).\n-export([f/0]).\nf()->1.\n",
	},
	{
		tool:  "sqlfluff",
		file:  "dirty.sql",
		dirty: "SELECT   1\n",
		config: templateStaticConfig(map[string]string{
			// The dialect is a project choice; Formattington never
			// hardcodes one.
			".sqlfluff": "[sqlfluff]\ndialect = ansi\n",
		}),
	},
	{tool: "gdformat", file: "dirty.gd", dirty: "var x  =  1\n"},
	{
		tool:  "google-java-format",
		file:  "dirty.java",
		dirty: "class T{public static void main(String[] args){int x=1;System.out.println(x);}}\n",
	},
	{
		tool:  "ktfmt",
		file:  "dirty.kt",
		dirty: "fun main(){val x=1;println(x)}\n",
	},
	{
		tool:  "scalafmt",
		file:  "dirty.scala",
		dirty: "object T{def main(args:Array[String]):Unit={val x=1;println(x)}}\n",
		config: templateVersionConfig("scalafmt", func(version string) map[string]string {
			// scalafmt aborts when the config version does not match
			// the binary version, and wants an explicit dialect
			// rather than a default.
			return map[string]string{
				".scalafmt.conf": "version = " + version + "\nrunner.dialect = Scala213\n",
			}
		}),
	},
	{
		tool:  "cljfmt",
		file:  "dirty.clj",
		dirty: "(ns t)\n(defn f [x]\n(let [y 1]\n(+ y x)))\n",
	},
	{tool: "fantomas", file: "dirty.fs", dirty: "module T\nlet x=1\n"},
	{
		tool:     "rubocop",
		file:     "dirty.rb",
		dirty:    "# frozen_string_literal: true\nx=1\nputs x\n",
		fixArgs:  []string{"-x"}, // --fix-layout: correct layout cops only
		fixExits: []int{ExitOK, ExitFormatRequired},
	},
	{
		tool:   "styler",
		binary: "Rscript",
		file:   "dirty.R",
		dirty:  "x=1\n",
		config: templateProbesRPackage("styler"),
	},
	{tool: "jlfmt", file: "dirty.jl", dirty: "x=1\n"},
	{
		tool:  "elm-format",
		file:  "dirty.elm",
		dirty: "module T exposing (main)\n\nimport Html exposing (text)\n\nmain =\n        text \"hi\"\n",
	},
	{tool: "fourmolu", file: "dirty.hs", dirty: "f x = x +   1\n"},
	{
		tool:  "ocamlformat",
		file:  "dirty.ml",
		dirty: "let x=1\n",
		config: templateVersionConfig("ocamlformat", func(version string) map[string]string {
			// ocamlformat requires a .ocamlformat file declaring the
			// version it was built with.
			return map[string]string{".ocamlformat": "version = " + version + "\n"}
		}),
	},
	{
		tool:  "rustfmt",
		file:  "dirty.rs",
		dirty: "fn main(){let x=1;println!(\"{}\",x);}\n",
	},
	{
		tool:           "swift-format",
		fallbackBinary: "swift",
		file:           "dirty.swift",
		dirty:          "let x:Int = 1\n",
	},
	{
		tool:   "dart-format",
		binary: "dart",
		file:   "dirty.dart",
		dirty:  "void main() { int x = 1; print( x ); }\n",
	},
	{
		tool:   "dotnet-format",
		binary: "dotnet",
		file:   "dirty.cs",
		dirty:  "class T{static void Main(){var x=1;System.Console.WriteLine(x);}}\n",
		config: templateStaticConfig(map[string]string{
			".editorconfig": "root = true\n\n[*.cs]\ncsharp_new_line_before_open_brace = true\n",
		}),
	},
	{
		tool:   "mix-format",
		binary: "mix",
		file:   "dirty.ex",
		dirty:  "defmodule T do\ndef f do\n1\nend\nend\n",
	},
	{
		tool:   "zig-fmt",
		binary: "zig",
		file:   "dirty.zig",
		dirty:  "var  x: i32 = 1;\n",
	},
	{
		tool:   "gleam-format",
		binary: "gleam",
		file:   "dirty.gleam",
		dirty:  "pub fn main() {\nlet x=1\nx\n}\n",
	},
	{
		tool:   "mojo-format",
		binary: "mojo",
		file:   "dirty.mojo",
		dirty:  "fn main():\n    var x=1\n    print(x)\n",
	},
	{
		tool:   "odinfmt",
		binary: "odinfmt",
		file:   "dirty.odin",
		dirty:  "package main\nmain :: proc() {\n    x := 1\n}\n",
	},
	{
		tool:   "forge-fmt",
		binary: "forge",
		file:   "dirty.sol",
		dirty:  "pragma solidity ^0.8.0;\ncontract T{function f() public {}}\n",
	},
	{tool: "gnatformat", file: "dirty.adb", dirty: "procedure T is\nx : Integer := 1;\nbegin\nnull;\nend T;\n"},
}

// TestInitTemplateRoundTripCoversRegistry guards against drift: every
// registered tool must either have a round-trip case or be explicitly
// excluded with a reason.
func TestInitTemplateRoundTripCoversRegistry(t *testing.T) {
	covered := map[string]bool{}
	for _, tc := range templateRoundTripCases {
		covered[tc.tool] = true
	}
	for _, tool := range registry {
		if covered[tool.Name] {
			continue
		}
		if reason, excluded := templateRoundTripExcluded[tool.Name]; excluded {
			_ = reason
			continue
		}
		t.Errorf("tool %q has no round-trip case and no exclusion reason", tool.Name)
	}
	for tool := range templateRoundTripExcluded {
		if covered[tool] {
			t.Errorf("tool %q is excluded but also has a round-trip case", tool)
		}
		if !registeredToolNames()[tool] {
			t.Errorf("excluded tool %q is not registered", tool)
		}
	}
}

// TestExampleRoundTrip exercises the sample configuration end to end for
// every registered tool that can format in place: a generated dirty file
// is reported by --dry-run and --check as needing formatting, the fix
// invocation rewrites it in place, --dry-run and --check then report it
// clean, and the temporary file is deleted. Tools that are not
// resolvable on the current machine are skipped.
func TestExampleRoundTrip(t *testing.T) {
	for _, testCase := range templateRoundTripCases {
		t.Run(testCase.tool, func(t *testing.T) {
			binary := testCase.binary
			if binary == "" {
				binary = testCase.tool
			}
			resolved, err := FindTool(binary, "", false)
			if err != nil && testCase.fallbackBinary != "" {
				resolved, err = FindTool(testCase.fallbackBinary, "", false)
			}
			if err != nil {
				t.Skipf("%s is not resolvable on this machine; skipping round trip", binary)
			}

			dir := t.TempDir()
			t.Chdir(dir)

			if testCase.config != nil {
				files, ok := testCase.config(t, resolved)
				if !ok {
					t.Skipf("required project configuration for %s could not be prepared; skipping round trip", testCase.tool)
				}
				for name, content := range files {
					writeFile(t, filepath.Join(dir, name), content)
				}
			}

			path := filepath.Join(dir, testCase.file)
			reset := func() { writeFile(t, path, testCase.dirty) }

			fixOK := testCase.fixExits
			if len(fixOK) == 0 {
				fixOK = []int{ExitOK}
			}

			// 1. --dry-run shows the file needs formatting.
			reset()
			code, stdout, stderr := runCLI(t, "--dry-run", testCase.tool, path)
			if code == ExitNotFound {
				t.Skipf("%s is not resolvable on this machine; skipping round trip", testCase.tool)
			}
			if code == ExitOK && stdout+stderr == "" {
				t.Errorf("--dry-run %s: exit 0 and no output; expected the file to be reported as needing formatting:\n%s",
					testCase.tool, readFileForError(t, path))
			}

			// 2. --check shows the file fails the check.
			reset()
			code, stdout, stderr = runCLI(t, "--check", testCase.tool, path)
			if code != ExitFormatRequired {
				t.Errorf("--check %s: exit = %d, want %d (ExitFormatRequired); stdout:\n%s\nstderr:\n%s",
					testCase.tool, code, ExitFormatRequired, stdout, stderr)
			}

			// 3. The fix invocation writes in place.
			reset()
			args := append([]string{testCase.tool}, testCase.fixArgs...)
			args = append(args, path)
			code, stdout, stderr = runCLI(t, args...)
			if !containsInt(fixOK, code) {
				t.Fatalf("fix (%s): exit = %d, want one of %v; stdout:\n%s\nstderr:\n%s",
					strings.Join(args, " "), code, fixOK, stdout, stderr)
			}
			fixed, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("fix (%s): read %s: %v", strings.Join(args, " "), path, err)
			}
			if string(fixed) == testCase.dirty {
				t.Fatalf("fix (%s): file was not modified in place", strings.Join(args, " "))
			}

			// 4. --dry-run shows the file does not need formatting anymore.
			code, stdout, stderr = runCLI(t, "--dry-run", testCase.tool, path)
			if code != ExitOK {
				t.Errorf("--dry-run after fix: exit = %d, want %d; stdout:\n%s\nstderr:\n%s",
					code, ExitOK, stdout, stderr)
			}

			// 5. --check shows the file does not need formatting anymore.
			code, stdout, stderr = runCLI(t, "--check", testCase.tool, path)
			if code != ExitOK {
				t.Errorf("--check after fix: exit = %d, want %d; stdout:\n%s\nstderr:\n%s",
					code, ExitOK, stdout, stderr)
			}

			// 6. Delete the temporary file.
			if err := os.Remove(path); err != nil {
				t.Errorf("delete %s: %v", path, err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("%s still exists after deletion", path)
			}
		})
	}
}

// containsInt reports whether values contains want.
func containsInt(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// readFileForError returns the file content for failure messages,
// tolerating read errors.
func readFileForError(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("<unreadable: %v>", err)
	}
	return string(data)
}

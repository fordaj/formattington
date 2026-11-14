package formattington

import (
	"fmt"
	"io"
	"sort"
)

// usageLine is the one-line usage summary printed with usage errors.
const usageLine = "usage: formattington [-h] [--version] [--check | --dry-run] [--series] [--verbose] [--config PATH]"

const helpEpilog = `Language aliases (same --check/--dry-run and forwarded-argument semantics as
formatter names, listed in Stack Overflow Developer Survey 2026 popularity
order; TOML and Markdown tooling are unranked and sort last): md/markdown use prettier (Markdown formatting is provided by Prettier; .md and .markdown
files are covered; Markdown style is linted with the markdownlint tool);
html uses prettier; sql uses sqlfluff (SQL lint/format via SQLFluff; configure
the dialect in the project or forward --dialect yourself). python/py use
black; micropython uses black (MicroPython is a Python 3 subset, so it is
formatted by the Python formatter — fixtures verified against black 26.5.1 and
ruff, including @micropython.native/viper and inline asm; statement-level asm
labels cannot be parsed by either); bash uses shfmt. java uses
google-java-format (Java formatting via google-java-format; whitespace-only
diffs by default via --cleanup=false). csharp/vbnet use dotnet-format (C#/VB.NET
whitespace formatting via "dotnet format whitespace --folder . --include
<files>"; .NET SDK >= 6 required; --verify-no-changes exit 2 is reported as 1).
ps1 uses powershell (PSScriptAnalyzer linting of PowerShell scripts); c/cpp/c++
use clang-format; rust uses rustfmt (Rust formatting via rustfmt; edition
detected per file from the nearest Cargo.toml, nearest workspace.package
edition honored). php uses php-cs-fixer (PHP formatting; "php-cs-fixer check"
is non-modifying, exit 8 is reported as 1). go uses gofmt (Go formatting via
gofmt; goimports additionally fixes imports, so use it via the goimports name
or a config command for import-aware formatting); kotlin uses ktfmt (Kotlin
formatting via ktfmt; pass --google-style/--kotlinlang-style verbatim as
needed). lua uses stylua (Lua formatting via StyLua); swift uses swift-format
(Swift formatting — tries standalone swift-format, falls back to "swift format"
from Swift 6+ toolchains). ruby uses rubocop (Ruby linting with layout cops;
every mode lints — pass -x for layout-only fixes; findings exit 1). dart uses
dart-format (Dart formatting via "dart format" from the Dart SDK; invoked as
"dart format"). r uses styler (R formatting via the styler package through
Rscript; check mode uses dry = "fail"). nix uses nixfmt (Nix formatting via
nixfmt, RFC 166 "rfc style"); gdscript uses gdformat (Godot 4 GDScript
formatting via gdformat, the gdtoolkit package; "pipx install
gdtoolkit==4.*"). groovy uses npm-groovy-lint (Groovy layout formatting; needs
Node >= 18 and Java 11+; enable the "format" preset in .groovylintrc or layout
findings are not reported). perl uses perltidy (Perl formatting; PATH-only and
GPL-2.0+; check mode uses "perltidy --assert-tidy"). zig uses zig-fmt (Zig
formatting via "zig fmt" from the Zig toolchain). elixir uses mix-format (Elixir
formatting via mix format from the Elixir/OTP installation; --no-compile is
passed by default). haskell uses fourmolu (Haskell formatting via fourmolu).
scala uses scalafmt (Scala formatting; PATH-only — the native binaries cannot
honor the mandatory .scalafmt.conf version pin, so Formattington does not
bundle them). fortran uses fprettify (Fortran formatting; PATH-only and
GPL-3.0+; check mode is synthesized via "fprettify --diff"). erlang uses erlfmt
(Erlang formatting via erlfmt; requires Erlang/OTP 21+). julia uses jlfmt
(Julia formatting via the JuliaFormatter.jl app; install with Pkg.Apps.add).
clojure uses cljfmt (Clojure formatting via cljfmt; subcommand-shaped,
"cljfmt check" exits 1 on unformatted files). gleam uses gleam-format
(Gleam formatting via "gleam format" from the Gleam toolchain). ada uses
gnatformat (Ada formatting via GNATformat; when no .gpr project exists in the
cwd, --no-project is passed automatically for standalone files). fsharp uses
fantomas (F# formatting via fantomas; needs the .NET SDK >= 8; exit 99 means
formatting needed, reported as 1). ocaml uses ocamlformat (OCaml formatting via
ocamlformat; install from opam). solidity uses forge-fmt (Solidity formatting
via "forge fmt" from Foundry). mojo uses mojo-format (Mojo formatting via
"mojo format"; check mode is synthesized by copying to a temp dir — the user
tree is never modified; Linux glibc >= 2.34 and macOS arm64 only). odin uses
odinfmt (Odin formatting via odinfmt; check mode is synthesized by
byte-comparing stdout against the original). elm uses elm-format (Elm formatting via
elm-format; zero-configuration). toml uses taplo (TOML formatting via taplo fmt).

markdownlint lints Markdown files; it never rewrites files without an
explicit "--fix" argument, so every mode lints (findings exit 1, no
findings exit 0). Configuration is discovered by the tool itself (for
example .markdownlint.json in the working directory).

With no file arguments, formattington discovers each formatter's matching
files recursively from the current directory (skipping hidden directories and
__pycache__, node_modules, vendor, venv, dist, build, and formatters/) and formats
them in batches.

Bundled formatters live under formatters/<name>/v<version>/<platform>/ or, in
the "full" release binaries, inside the executable itself; use --verbose
(or set FORMATTINGTON_VERBOSE=1) to show the resolved executable.

With 'all', formatters run concurrently by default; use --series (before or
after 'all', e.g. 'formattington --series all' or 'formattington all
--series') to run them sequentially in Stack Overflow Developer Survey 2026
popularity order (prettier, sqlfluff, black, shfmt, google-java-format,
dotnet-format, powershell, clang-format, rustfmt, php-cs-fixer, gofmt,
goimports, ktfmt, stylua, swift-format, rubocop, dart-format, styler, nixfmt,
gdformat, npm-groovy-lint, perltidy, zig-fmt, mix-format, fourmolu, scalafmt,
fprettify, erlfmt, jlfmt, cljfmt, gleam-format, gnatformat, fantomas,
ocamlformat, forge-fmt, mojo-format, odinfmt, elm-format, markdownlint, taplo).
Exit code is the last non-zero code
in that order (missing tool counts as 127, printed to stderr, and does not
prevent remaining formatters from running); 0 only if all succeed.

Configuration file for 'all': a JSON file with a 'commands' array of argv
arrays, e.g. {"commands": [["black", "--check", "src"], ["prettier",
"docs/**/*.md"]]}. Element 0 of each command is a formatter name or language
alias; the rest is forwarded verbatim. Commands run concurrently by default
and sequentially in file order with --series; --check/--dry-run/--verbose
propagate to each command and the exit code is the last non-zero code in file
order (missing tool is 127 per command and does not stop remaining commands;
0 only if all succeed). An empty 'commands': [] runs nothing and exits 0.
Discovery precedence (highest first): --config PATH (before or after 'all'),
then FORMATTINGTON_CONFIG_PATH, then ./.formattington.json in the current
directory (cwd only, no parent-directory search), then no config (full
registry 'all' as before). Shell strings, object-form entries, and bare
top-level arrays are rejected; unknown top-level keys are rejected.
Missing/invalid config exits 2 with a 'formattington: ...' message naming the
file. Single-tool runs ignore the config file and environment variable;
--config before a single formatter is a usage error (exit 2). Relative paths
are resolved against the current directory.

'formattington list' prints a table of every registered tool with the
version of the bundle Formattington would resolve for the host platform and
the version found on PATH; a side shows '-' when it does not exist. The PATH
version is probed by invoking the resolved tool (its version flag, or the
equivalent the tool supports; gofmt and goimports are probed via 'go
version'), so the probe runs at list time and never modifies files. 'list'
takes no arguments or mode flags; the exit code is always 0.`

// printHelp writes the help text.
func printHelp(out io.Writer) {
	fmt.Fprintf(out, "%s\n\n", usageLine)
	fmt.Fprintf(out, "One formatter interface for your entire project.\n\n")

	fmt.Fprintf(out, "options:\n")
	fmt.Fprintf(out, "  -h, --help       show this help message and exit\n")
	fmt.Fprintf(out, "      --version    show the version and exit\n")
	fmt.Fprintf(out, "      --check      report formatting changes without modifying files\n")
	fmt.Fprintf(out, "      --dry-run    validate without modifying files\n")
	fmt.Fprintf(out, "      --series     with 'all', run formatters sequentially instead of concurrently\n")
	fmt.Fprintf(out, "      --verbose    show which formatter executable was resolved\n")
	fmt.Fprintf(out, "      --config PATH with 'all', run commands from this JSON config file\n\n")

	fmt.Fprintf(out, "formatters:\n")
	for _, tool := range registry {
		fmt.Fprintf(out, "  %s\n", tool.Name)
	}
	fmt.Fprintf(out, "  all              run every registered formatter\n")
	fmt.Fprintf(out, "  list             show each registered tool's bundled and PATH versions\n\n")

	fmt.Fprintf(out, "language aliases:\n")
	for _, alias := range sortedAliasNames() {
		fmt.Fprintf(out, "  %-8s -> %s\n", alias, languageAliases[alias])
	}
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "%s\n", helpEpilog)
}

// toolSurveyRank maps a tool name to the Stack Overflow Developer Survey
// 2026 rank ("Programming, scripting, and markup languages") of the
// best-ranked language it covers, per AGENTS.md "Presentation order". Tools
// with no survey row are absent and sort last.
var toolSurveyRank = map[string]int{
	"prettier":           1,
	"sqlfluff":           2,
	"black":              4,
	"shfmt":              5,
	"google-java-format": 7,
	"dotnet-format":      8,
	"powershell":         9,
	"clang-format":       10,
	"rustfmt":            12,
	"php-cs-fixer":       13,
	"gofmt":              14,
	"ktfmt":              15,
	"stylua":             16,
	"swift-format":       18,
	"rubocop":            19,
	"dart-format":        21,
	"styler":             22,
	"nixfmt":             23,
	"gdformat":           24,
	"npm-groovy-lint":    27,
	"perltidy":           28,
	"zig-fmt":            30,
	"mix-format":         33,
	"fourmolu":           35,
	"scalafmt":           36,
	"fprettify":          38,
	"erlfmt":             39,
	"jlfmt":              40,
	"cljfmt":             41,
	"gleam-format":       42,
	"gnatformat":         43,
	"fantomas":           44,
	"ocamlformat":        45,
	"forge-fmt":          48,
	"mojo-format":        49,
	"odinfmt":            50,
	"elm-format":         51,
}

// unrankedToolRank sorts tools with no survey row (markdownlint, taplo)
// after every ranked tool.
const unrankedToolRank = 1000

// sortedAliasNames returns the language alias names ordered by their
// backend tool's survey rank (AGENTS.md "Presentation order"); aliases
// sharing a rank are alphabetical, and unranked aliases (toml) sort last,
// alphabetical among themselves.
func sortedAliasNames() []string {
	names := make([]string, 0, len(languageAliases))
	for alias := range languageAliases {
		names = append(names, alias)
	}
	sort.Slice(names, func(i, j int) bool {
		ri, rj := aliasRank(names[i]), aliasRank(names[j])
		if ri != rj {
			return ri < rj
		}
		return names[i] < names[j]
	})
	return names
}

// aliasRank returns the survey rank of the language alias's backend tool.
func aliasRank(alias string) int {
	if r, ok := toolSurveyRank[languageAliases[alias]]; ok {
		return r
	}
	return unrankedToolRank
}

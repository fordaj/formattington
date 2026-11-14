package formattington

import (
	"fmt"
	"io"
	"sort"
)

// usageLine is the one-line usage summary printed with usage errors.
const usageLine = "usage: formattington [-h] [--version] [--check | --dry-run] [--series] [--verbose] [--config PATH]"

const helpEpilog = `Language aliases (same --check/--dry-run and forwarded-argument semantics as
formatter names): c/cpp/c++ use clang-format; bash uses shfmt; python/py use
black; md/markdown use prettier (Markdown formatting is provided by Prettier;
.md and .markdown files are covered; Markdown style is linted with the
markdownlint tool); html uses prettier; ps1 uses powershell
(PSScriptAnalyzer linting of PowerShell scripts); toml uses taplo (TOML
formatting via taplo fmt); go uses gofmt (Go formatting via gofmt; goimports
additionally fixes imports, so use it via the goimports name or a config
command for import-aware formatting); micropython uses black (MicroPython is a
Python 3 subset, so it is formatted by the Python formatter — fixtures
verified against black 26.5.1 and ruff, including @micropython.native/viper
and inline asm; statement-level asm labels cannot be parsed by either); lua uses stylua (Lua formatting via
StyLua); nix uses nixfmt (Nix formatting via nixfmt, RFC 166 "rfc style"); php uses
php-cs-fixer (PHP formatting; "php-cs-fixer check" is non-modifying, exit 8
is reported as 1). perl uses perltidy (Perl formatting; PATH-only and GPL-2.0+; check mode
uses "perltidy --assert-tidy"). fortran uses fprettify (Fortran formatting;
PATH-only and GPL-3.0+; check mode is synthesized via "fprettify --diff"). erlang
uses erlfmt (Erlang formatting via erlfmt; requires Erlang/OTP 21+). sql uses
sqlfluff (SQL lint/format via SQLFluff; configure the dialect in the project
or forward --dialect yourself). gdscript uses gdformat (Godot 4 GDScript
formatting via gdformat, the gdtoolkit package; "pipx install gdtoolkit==4.*").
java uses google-java-format (Java formatting via google-java-format; whitespace-only
diffs by default via --cleanup=false). kotlin uses ktfmt (Kotlin formatting via
ktfmt; pass --google-style/--kotlinlang-style verbatim as needed). scala uses
scalafmt (Scala formatting; PATH-only — the native binaries cannot honor the
mandatory .scalafmt.conf version pin, so Formattington does not bundle them).
groovy uses npm-groovy-lint (Groovy layout formatting; needs Node >= 18 and Java 11+;
enable the "format" preset in .groovylintrc or layout findings are not reported). clojure uses cljfmt (Clojure formatting via cljfmt; subcommand-shaped,
"cljfmt check" exits 1 on unformatted files). fsharp uses fantomas (F# formatting via
fantomas; needs the .NET SDK >= 8; exit 99 means formatting needed, reported as 1).

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
--series') to run them sequentially in shfmt, clang-format, black, prettier,
powershell, markdownlint, taplo, gofmt, goimports, stylua, nixfmt, php-cs-fixer, perltidy, fprettify, erlfmt, sqlfluff, gdformat, google-java-format, ktfmt, scalafmt, npm-groovy-lint, cljfmt, fantomas order. Exit code is the
last non-zero code
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
are resolved against the current directory.`

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
	fmt.Fprintf(out, "  all              run every registered formatter\n\n")

	fmt.Fprintf(out, "language aliases:\n")
	for _, alias := range sortedAliasNames() {
		fmt.Fprintf(out, "  %-8s -> %s\n", alias, languageAliases[alias])
	}
	fmt.Fprintf(out, "\n")

	fmt.Fprintf(out, "%s\n", helpEpilog)
}

// sortedAliasNames returns the language alias names in sorted order.
func sortedAliasNames() []string {
	names := make([]string, 0, len(languageAliases))
	for alias := range languageAliases {
		names = append(names, alias)
	}
	sort.Strings(names)
	return names
}

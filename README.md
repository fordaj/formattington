# Formattington

**One interface for your project's formatting and code-quality tools.**

Formattington brings code formatters, linters, and other automated quality tools together under one roof. Python, C++, JavaScript, Bash, Markdown, and more — one command to keep multidisciplinary projects consistently formatted and maintained.

## Why Formattington?

Modern projects rarely use just one language or quality tool. Each language brings its own formatter, linter, configuration, command-line options, and installation requirements.

Formattington provides a common interface over these tools, so you can spend less time remembering individual commands and more time building things.

- **One interface** — run project tools through a single command.
- **Multi-language** — handle mixed-language repositories without juggling individual tools.
- **Flexible selection** — run every configured tool, choose a specific tool, or target a language.
- **Multiple operations** — format, check, lint, and automatically fix issues where supported.
- **Sequential or parallel execution** — choose whether tools run concurrently or in sequence.
- **Existing configurations** — reuse native tool configuration files.
- **Extensible** — integrate established tools without reimplementing their functionality.
- **Cross-platform ambitions** — designed for projects spanning Windows, macOS, and Linux.

## Usage

### Run all configured tools

```bash
formattington all
```

Run all configured formatters against their matching files. This is the default strategy: formatters run concurrently (via a thread pool), so independent tools overlap.

A missing or failing formatter does not prevent the others from running; its failure is reported on stderr and reflected in the exit code (see [Exit codes](#exit-codes)).

Output from different formatters may interleave in this mode.

### Run tools sequentially

```bash
formattington all --series
```

Run formatters one after another (prettier, sqlfluff, black, shfmt, google-java-format, dotnet-format, powershell, clang-format, rustfmt, php-cs-fixer, gofmt, goimports, ktfmt, stylua, swift-format, rubocop, dart-format, styler, nixfmt, gdformat, npm-groovy-lint, perltidy, zig-fmt, mix-format, fourmolu, scalafmt, fprettify, erlfmt, jlfmt, cljfmt, gleam-format, gnatformat, fantomas, ocamlformat, forge-fmt, mojo-format, odinfmt, elm-format, markdownlint, taplo order — the Stack Overflow Developer Survey 2026 popularity order) instead of concurrently, for ordered output.

The flag is also accepted before `all`:

```bash
formattington --series all
formattington --check --series all
```

`--check` and `--dry-run` keep their usual meaning (they go before `all`): in check mode every formatter still reports, whether you use the concurrent default, `--series`, or both flags together.

### Choose a specific tool

```bash
formattington prettier
formattington sqlfluff
formattington black
formattington shfmt
formattington google-java-format
formattington dotnet-format
formattington powershell
formattington clang-format
formattington rustfmt
formattington php-cs-fixer
formattington gofmt
formattington goimports
formattington ktfmt
formattington stylua
formattington swift-format
formattington rubocop
formattington dart-format
formattington styler
formattington nixfmt
formattington gdformat
formattington npm-groovy-lint
formattington perltidy
formattington zig-fmt
formattington mix-format
formattington fourmolu
formattington scalafmt
formattington fprettify
formattington erlfmt
formattington jlfmt
formattington cljfmt
formattington gleam-format
formattington gnatformat
formattington fantomas
formattington ocamlformat
formattington forge-fmt
formattington mojo-format
formattington odinfmt
formattington elm-format
formattington markdownlint
formattington taplo
```

Pass additional arguments through to the underlying tool:

```bash
formattington prettier --check
formattington clang-format --dry-run
formattington shfmt -d
```

Arguments are forwarded to the selected tool, subject to Formattington's own argument handling.

### File discovery

When you pass no file arguments (for example, `formattington shfmt`), Formattington discovers matching files recursively from the current directory and passes them to the formatter in batches — so `formattington shfmt` formats every `.sh`/`.bash` file, `formattington clang-format` formats every C/C++ file, and so on. Discovered files are passed as discrete arguments, so paths with spaces are handled safely.

Discovery skips hidden directories and the usual generated/vendored locations (`__pycache__`, `node_modules`, `vendor`, `venv`, `dist`, `build`, and the bundled-formatters directory `formatters/`). If you pass explicit file arguments (for example, `formattington shfmt myfile.sh`), discovery is skipped and only those files are processed. Informational passthroughs (`--help`, `--version`) never trigger discovery.

### Run by language or file type

```bash
formattington html
formattington markdown
formattington md
formattington sql
formattington micropython
formattington py
formattington python
formattington bash
formattington java
formattington csharp
formattington vbnet
formattington ps1
formattington c
formattington c++
formattington cpp
formattington rust
formattington php
formattington go
formattington kotlin
formattington lua
formattington swift
formattington ruby
formattington dart
formattington r
formattington nix
formattington gdscript
formattington groovy
formattington perl
formattington zig
formattington elixir
formattington haskell
formattington scala
formattington fortran
formattington erlang
formattington julia
formattington clojure
formattington gleam
formattington ada
formattington fsharp
formattington ocaml
formattington solidity
formattington mojo
formattington odin
formattington elm
formattington toml
```

Language selection lets you target the relevant files without having to remember which tool handles each extension; aliases are listed in Stack Overflow Developer Survey 2026 popularity order (unranked TOML last):

- `md`/`markdown`/`html` resolve to `prettier`
- `sql` resolves to `sqlfluff`
- `python`/`py`/`micropython` resolve to `black`
- `bash` resolves to `shfmt`
- `java` resolves to `google-java-format`
- `csharp`/`vbnet` resolve to `dotnet-format`
- `ps1` resolves to `powershell`
- `c`/`cpp`/`c++` resolve to `clang-format`
- `rust` resolves to `rustfmt`
- `php` resolves to `php-cs-fixer`
- `go` resolves to `gofmt`
- `kotlin` resolves to `ktfmt`
- `lua` resolves to `stylua`
- `swift` resolves to `swift-format`
- `ruby` resolves to `rubocop`
- `dart` resolves to `dart-format`
- `r` resolves to `styler`
- `nix` resolves to `nixfmt`
- `gdscript` resolves to `gdformat`
- `groovy` resolves to `npm-groovy-lint`
- `perl` resolves to `perltidy`
- `zig` resolves to `zig-fmt`
- `elixir` resolves to `mix-format`
- `haskell` resolves to `fourmolu`
- `scala` resolves to `scalafmt`
- `fortran` resolves to `fprettify`
- `erlang` resolves to `erlfmt`
- `julia` resolves to `jlfmt`
- `clojure` resolves to `cljfmt`
- `gleam` resolves to `gleam-format`
- `ada` resolves to `gnatformat`
- `fsharp` resolves to `fantomas`
- `ocaml` resolves to `ocamlformat`
- `solidity` resolves to `forge-fmt`
- `mojo` resolves to `mojo-format`
- `odin` resolves to `odinfmt`
- `elm` resolves to `elm-format`
- `toml` resolves to `taplo`

Use `goimports` when imports should also be fixed (for example `formattington goimports` or a config command listing `goimports`). Aliases accept the same `--check`/`--dry-run` flags (before the alias) and forwarded arguments (after it) as formatter names. Unrecognized words are a usage error (exit 2).

### Run with a configuration file (`.formattington.json`)

Instead of always running every formatter, `formattington all` can run an exact list of commands declared in a JSON configuration file. This is useful for mixed-language projects that want to version-control which tools and arguments they use:

```json
{
  "commands": [
    ["black", "--check", "src"],
    ["prettier", "docs/**/*.md"],
    ["shfmt", "-i", "2"]
  ]
}
```

Save it as `./.formattington.json` and run:

```bash
formattington all
```

or point at any file with:

```bash
formattington all --config path/to/config.json
formattington --config path/to/config.json all
formattington --check all --config path/to/config.json
```

Each entry in `commands` is an argv list: element 0 is a formatter name (`prettier`, `sqlfluff`, `black`, `shfmt`, `google-java-format`, `dotnet-format`, `powershell`, `clang-format`, `rustfmt`, `php-cs-fixer`, `gofmt`, `goimports`, `ktfmt`, `stylua`, `swift-format`, `rubocop`, `dart-format`, `styler`, `nixfmt`, `gdformat`, `npm-groovy-lint`, `perltidy`, `zig-fmt`, `mix-format`, `fourmolu`, `scalafmt`, `fprettify`, `erlfmt`, `jlfmt`, `cljfmt`, `gleam-format`, `gnatformat`, `fantomas`, `ocamlformat`, `forge-fmt`, `mojo-format`, `odinfmt`, `elm-format`, `markdownlint`, `taplo`) or a language alias (`md`/`markdown`/`html`, `sql`, `python`/`py`/`micropython`, `bash`, `java`, `csharp`/`vbnet`, `ps1`, `c`/`cpp`/`c++`, `rust`, `php`, `go`, `kotlin`, `lua`, `swift`, `ruby`, `dart`, `r`, `nix`, `gdscript`, `groovy`, `perl`, `zig`, `elixir`, `haskell`, `scala`, `fortran`, `erlang`, `julia`, `clojure`, `gleam`, `ada`, `fsharp`, `ocaml`, `solidity`, `mojo`, `odin`, `elm`, `toml`); the remaining elements are forwarded verbatim to that formatter, exactly as arguments after the formatter name on the CLI (for example, `["black", "--check", "src"]` behaves like `formattington black --check src`). Arguments stay as discrete list entries, so paths with spaces are safe. When a command passes no explicit files, file discovery works as for single-tool runs. A sample is provided as `.formattington.example.json` (it is not auto-discovered). It showcases every registered tool with its most commonly used formatting invocation — one command each, in registration order — and the test suite keeps it honest: `internal/formattington/example_test.go` generates a `dirty.<ext>` fixture per tool, verifies that `--dry-run` and `--check` report the file as needing formatting, applies the fix (the default in-place mode, or `--fix`/`-x` for markdownlint and rubocop), verifies that both then report it clean, and deletes the temporary file. Tools that cannot format in place (`powershell`, `npm-groovy-lint`) are not round-tripped, and any tool that is not resolvable on the current machine is skipped; the example-validation tests always run.

Config discovery precedence (highest first):

| Priority | Source                                           | Notes                                                                                                                                              |
| -------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1        | `--config <path>` (before or after `all`)        | Explicit path, relative paths resolve against the current directory; `--config` after `all` (e.g. `formattington all --config x.json`) is honored. |
| 2        | `FORMATTINGTON_CONFIG_PATH` environment variable | Explicit path; an empty value counts as unset.                                                                                                     |
| 3        | `./.formattington.json`                          | Auto-discovered in the current directory only (no parent-directory or user-home search).                                                           |
| 4        | No config file                                   | `all` runs the full formatter registry as before (backwards compatible).                                                                           |

When a config is in effect, `all` runs exactly the listed commands. With no config, `all` keeps its current behavior (every registered tool). The config only changes what `all` means: single-tool runs ignore the file and the environment variable, and `--config` before a single formatter (e.g. `formattington --config x.json black`) is a usage error (exit 2). `--config` after a single formatter is forwarded to that formatter, so native flags such as Prettier's `--config` keep working.

Global flags compose as with a full-registry `all`: `--check`/`--dry-run` and `--verbose` (or `FORMATTINGTON_VERBOSE=1`) propagate to every config command. Config commands run concurrently by default; use `--series` (before or after `all`) to run them sequentially in file order. The exit code is the last non-zero code in file order, or `0` only if every command succeeds. A missing formatter executable counts as `127` for that command and does not prevent the remaining commands from running.

Empty `commands: []` is valid; `all` then runs nothing and exits `0`.

Shell-string entries (`"black --check"`), object-form entries (`{"tool": "black", "args": [...]}`), and bare top-level arrays (`[["black"]]`) are rejected, as are unknown top-level keys (only `commands` is supported): the validator exits `2` with a `formattington: ...` message naming the file and the offending entry rather than misinterpreting them. Missing files, malformed JSON, and schema violations (missing/non-list `commands`, non-list/non-string/empty entries, unknown formatter/alias in element 0) all exit `2` the same way. A present but malformed `./.formattington.json` also exits `2` and does not silently fall back to the full registry.

### List tool availability

```bash
formattington list
```

Prints a pipe-separated table of every registered tool with two version columns: the version of the bundle Formattington would resolve for the host platform, and the version found on `PATH`. Either side shows `-` when it does not exist:

```
| Tool         | Bundled Version | PATH Version |
| ------------ | --------------- | ------------ |
| shfmt        | 3.14.1          | 3.13.1       |
| powershell   | -               | -            |
```

- The bundled side is read from the bundle layout (`formatters/<name>/v<version>/<platform>/`, or the per-user cache directory the full release binary unpacks its embedded formatters into) and reports the newest bundle carrying a host-platform executable — the same copy `FindTool()` would select.
- The PATH side is resolved the same way the integrations resolve it (for example, `swift-format` falls back to `swift format`; `powershell` tries `pwsh` then `powershell`) and is probed by invoking the tool's version flag at list time (`--version` where supported; `gofmt`/`goimports` are probed via `go version <path>`, `zig` via `zig version`, `dotnet` via `dotnet format --version`, and `Rscript` is asked for the installed styler package version). A missing executable or an undeterminable version reports `-`. The probe never modifies files and is bounded by a short timeout.
- `list` takes no arguments or mode flags — `--check`, `--dry-run`, `--series`, and `--config` are usage errors — and exits `0`.

## Operations and capabilities

Formattington orchestrates tools that may support different operations.

| Operation | Purpose                                                          |
| --------- | ---------------------------------------------------------------- |
| Format    | Rewrite files into a consistent representation.                  |
| Check     | Identify formatting or style violations without modifying files. |
| Lint      | Analyze files for rule violations and report findings.           |
| Fix       | Automatically resolve issues supported by the underlying tool.   |

Not every tool supports every operation. Formattington should respect native tool capabilities rather than force every integration into a common feature set.

For example, `markdownlint --fix` can automatically correct supported Markdown violations, but cannot resolve every linting issue.

Formatting and linting are complementary: a file can be correctly formatted while still containing lint violations.

## Exit codes

Formattington preserves underlying formatter exit codes and uses a fixed scheme for its own failures:

| Code | Meaning                                                                                                                                                                                                                                       |
| ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0    | Success. Formatting applied (or nothing needed formatting); informational commands (`--help`, `--version`, `list`) also exit `0`.                                                                                                                                                                   |
| 1    | Formatting changes required in check mode.                                                                                                                                                                                                    |
| 2    | Usage error (for example, unknown formatter, exclusive `--check` and `--dry-run` given together, unexpected arguments parsed by Formattington, or a missing/invalid `--config` / `FORMATTINGTON_CONFIG_PATH` / `./.formattington.json` file). |
| 127  | Formatter executable not found (no bundled binary and nothing on `PATH`).                                                                                                                                                                     |

### Mode semantics

Formattington owns the mode flag before the formatter name; arguments after the formatter are forwarded verbatim.

| Formatter    | `--check` (before formatter)                                                       | `--dry-run` (before formatter)                                                                 | Default                                                                                                               |
| ------------ | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| prettier     | `prettier --check` — fail if any file needs formatting                             | `prettier --list-different` — list files that differ, do not modify (Prettier has no `--diff`) | `prettier --write` — write in place (unless `--write`/`-w`, `--check`/`-c`, or `--list-different`/`-l` already given) |
| sqlfluff     | `sqlfluff lint --disable-progress-bar` — exit 1 on violations (exit 2 = error, passed through)       | `sqlfluff lint --disable-progress-bar` (no diff mode since SQLFluff 3.x; same invocation as check)         | `sqlfluff fix --disable-progress-bar` — fix in place                                                        |
| black        | `black --check` — fail if any file needs formatting                                | `black --diff` — show diff, do not modify                                                      | `black` in place (no flag)                                                                                            |
| shfmt        | `shfmt -l` — list unformatted files; exit 1 if any listed, else 0                  | `shfmt -d` — print diff, do not modify                                                         | `shfmt -w` — write in place                                                                                           |
| google-java-format | `google-java-format --dry-run --set-exit-if-changed` — exit 1 when formatting is needed                 | `google-java-format --dry-run --set-exit-if-changed` (same invocation; there is no patch output)              | `google-java-format --replace` — format in place                                                                |
| dotnet-format | `dotnet format whitespace --verify-no-changes --folder . --include <files...>` — exit 2→1              | same as check (never writes)                                                                          | `dotnet format whitespace --folder . --include <files...>` — format in place                             |
| powershell   | PSScriptAnalyzer lint (no in-place formatting); findings exit 1                    | Same PSScriptAnalyzer lint invocation (check/dry-run are accepted for CLI uniformity)          | Same PSScriptAnalyzer lint invocation (findings exit 1)                                                               |
| clang-format | `clang-format --dry-run --Werror` — fail if any file needs formatting              | `clang-format --dry-run` — validate without modifying                                          | `clang-format -i` — format in place (unless `-i`/`--in-place` already given)                                          |
| rustfmt      | `rustfmt [--edition E] --check` — exit 1 when formatting is needed (also used for parse errors)     | `rustfmt [--edition E] --emit diff` — print a diff                                                    | `rustfmt [--edition E]` — format in place                                                                |
| php-cs-fixer | `php-cs-fixer check` — fail if any file needs fixing (native exit 8 is reported as 1)              | `php-cs-fixer check` (non-modifying; same invocation)                                             | `php-cs-fixer fix` — fix in place                                                                               |
| gofmt        | `gofmt -l` — list unformatted files; exit 1 if any listed, else 0                  | `gofmt -d` — display diffs instead of rewriting files                                          | `gofmt -w` — write result to the source file instead of stdout (format in place)                                      |
| goimports    | `goimports -l` — list files whose formatting differs; exit 1 if any listed, else 0 | `goimports -d` — display diffs instead of rewriting files                                      | `goimports -w` — write result to the source file instead of stdout (format in place, fixing imports)                  |
| ktfmt        | `ktfmt --dry-run --set-exit-if-changed` — exit 1 when formatting is needed                        | `ktfmt --dry-run` — list files that would change (upstream always exits 0; NOT a check)                        | `ktfmt` — format in place                                                                                       |
| stylua       | `stylua --check` — fail if any file needs formatting; files are not modified                   | `stylua --check --output-format=unified` — print unified diffs, do not modify                    | `stylua` — format in place                                                                                          |
| swift-format | `swift format lint --strict --ignore-unparsable-files` — exit 1 on findings                           | same as check (no diff mode)                                                                          | `swift format -i --ignore-unparsable-files` — format in place                                       |
| rubocop      | `rubocop` — lint posture (all modes; findings exit 1, config error exit 2)                        | `rubocop` (same invocation)                                                                           | `rubocop` — lint posture; pass `-x` for layout-only fixes, `-a`/`-A` for all auto-corrections                |
| dart-format  | `dart format --set-exit-if-changed -o none` — exit 1 when formatting is needed                        | same as check (never writes)                                                                          | `dart format` — format in place                                                                        |
| styler       | `styler::style_file(..., dry = "fail")` via Rscript — errors abort; reported as 1                    | `styler::style_file(..., dry = "fail")` (same invocation)                                             | `styler::style_file(..., dry = "off")` — format in place                                             |
| nixfmt       | `nixfmt --check` — fail if any file needs formatting (exit 1; also used for parse errors)       | `nixfmt --check` (no diff mode exists) — validate without modifying                               | `nixfmt` — format in place                                                                                       |
| gdformat     | `gdformat -c` — print "would reformat <file>" and exit 1 when reformatting is needed              | `gdformat -d` — print a diff, do not modify (implies --check)                                           | `gdformat` — format in place                                                                                |
| npm-groovy-lint | `npm-groovy-lint --failon error` — linter-flavored check (no writes; NOT a byte comparison)         | `npm-groovy-lint --failon error` (same invocation)                                                      | `npm-groovy-lint --format` — format in place (layout rules only)                                                |
| perltidy     | `perltidy --assert-tidy` — exit 0 when tidy, non-zero (reported as 1) otherwise                  | `perltidy -d` — print a diff, do not modify                                                       | `perltidy -b -bext=/` — format in place (backups suppressed)                                                      |
| zig-fmt      | `zig fmt --check` — exit 1 when any file would change                                                   | `zig fmt --check` (same invocation; there is no diff mode)                                                | `zig fmt` — format in place                                                                                |
| mix-format   | `mix format --check-formatted --no-compile` — exit non-zero (reported as 1) when formatting is needed  | same as check (never writes)                                                                          | `mix format --no-compile` — format in place                                                              |
| fourmolu     | `fourmolu --mode check` — print a unified diff (stderr) and exit 100 when formatting is needed        | `fourmolu --mode check` (same invocation)                                                            | `fourmolu --mode inplace` — format in place                                                               |
| scalafmt     | `scalafmt --test` — print a diff and exit 1 when formatting is needed                             | `scalafmt --test` (same invocation — this is scalafmt's diff mode)                                     | `scalafmt` — format in place                                                                                    |
| fprettify    | Synthesized: `fprettify --diff` — non-empty diff output means changes required (exit 1)          | `fprettify --diff` — print a diff, do not modify                                                  | `fprettify` — format in place                                                                                |
| erlfmt       | `erlfmt -c` — fail if any file needs formatting (exit 1)                                          | `erlfmt -c` (no diff mode exists) — validate without modifying                                      | `erlfmt -w` — format in place                                                                               |
| jlfmt        | `jlfmt --check` — exit 1 when formatting is needed (2 on error, passed through)                        | `jlfmt --diff` — print a unified diff; needs git on PATH, else degrades to --check                  | `jlfmt --inplace` — format in place                                                                        |
| cljfmt       | `cljfmt check` — print a unified diff and exit 1 when formatting is needed                          | `cljfmt check` (same invocation — this is cljfmt's diff mode)                                            | `cljfmt fix` — format in place                                                                            |
| gleam-format | `gleam format --check` — exit non-zero (Error::Format = 1) listing files that would change             | same as check (no diff mode)                                                                          | `gleam format` — format in place                                                                        |
| gnatformat   | `gnatformat [--no-project] --check` — exit 1 when formatting is needed                                  | same as check (no built-in diff; `--pipe` forwarded verbatim)                                             | `gnatformat [--no-project]` — format in place                                                                |
| fantomas     | `fantomas --check` — exit 99 when formatting is needed (reported as 1); 1 (error) passes through    | `fantomas --check` (same invocation; there is no diff mode)                                           | `fantomas` — format in place                                                                               |
| ocamlformat  | `ocamlformat --check` — exit 1 when formatting is needed (also used for errors; stderr disambiguates)  | `ocamlformat --check` (same invocation; there is no diff mode)                                       | `ocamlformat --inplace` — format in place                                                                  |
| forge-fmt    | `forge fmt --check` — exit 1 when formatting is required (0 clean; diff printed by default)             | same as check (no separate diff mode; `-r` for raw output)                                            | `forge fmt` — format in place                                                                               |
| mojo-format  | Synthesized: copy→`mojo format`→byte-compare — exit 1 when any file would change (tree never touched)  | same synthesis (no diff mode exists)                                                                  | `mojo format` — format in place                                                                        |
| odinfmt      | Synthesized: byte-compare `odinfmt <file>` stdout vs the original (one process per file)               | same synthesis (no diff mode exists)                                                                  | `odinfmt -w` — format in place (creates an `<file>_bk` backup first)                                      |
| elm-format   | `elm-format --validate` — print a JSON report and exit 1 when formatting is needed                  | `elm-format --validate` (same invocation; there is no diff mode)                                    | `elm-format --yes` — format in place (accept all prompts)                                                |
| markdownlint | `markdownlint` verbatim — never rewrites without an explicit `--fix`; every mode lints; findings exit 1 (no findings exit 0) | Same invocation (every mode lints) | Same invocation; findings exit 1 (`--fix` rewrites only when explicitly passed); configuration discovered natively (e.g. `.markdownlint.json`) |
| taplo        | `taplo fmt --check` — fail if any file needs formatting                            | `taplo fmt --diff` — print diff, do not modify                                                 | `taplo fmt` in place (no flag)                                                                                        |

These mode mappings are integration-specific. Future tools may support different operations or native flags, and their integrations should preserve those differences.

### Missing executables

When a formatter executable cannot be found, Formattington does not crash with a stack trace. It prints a single line to stderr (for example:

```text
formattington: Could not find 'shfmt'. Expected a bundled executable under formatters/shfmt/<version>/<platform>/shfmt or an installation on PATH.
```

) and exits `127`.

This happens identically in default, `--check`, and `--dry-run` modes, for `prettier`, `sqlfluff`, `black`, `shfmt`, `google-java-format`, `dotnet-format`, `powershell` (resolved as `pwsh`, then `powershell`, plus the PSScriptAnalyzer module), `clang-format`, `rustfmt`, `php-cs-fixer`, `gofmt`, `goimports`, `ktfmt`, `stylua`, `swift-format`, `rubocop`, `dart-format`, `styler`, `nixfmt`, `gdformat`, `npm-groovy-lint`, `perltidy`, `zig-fmt`, `mix-format`, `fourmolu`, `scalafmt`, `fprettify`, `erlfmt`, `jlfmt`, `cljfmt`, `gleam-format`, `gnatformat`, `fantomas`, `ocamlformat`, `forge-fmt`, `mojo-format`, `odinfmt`, `elm-format`, `markdownlint`, and `taplo`, and under `all` (the remaining formatters still run in both the concurrent default and `--series`).

### Exit-code aggregation with `all`

The exit code is the **last non-zero code in `prettier`, `sqlfluff`, `black`, `shfmt`, `google-java-format`, `dotnet-format`, `powershell`, `clang-format`, `rustfmt`, `php-cs-fixer`, `gofmt`, `goimports`, `ktfmt`, `stylua`, `swift-format`, `rubocop`, `dart-format`, `styler`, `nixfmt`, `gdformat`, `npm-groovy-lint`, `perltidy`, `zig-fmt`, `mix-format`, `fourmolu`, `scalafmt`, `fprettify`, `erlfmt`, `jlfmt`, `cljfmt`, `gleam-format`, `gnatformat`, `fantomas`, `ocamlformat`, `forge-fmt`, `mojo-format`, `odinfmt`, `elm-format`, `markdownlint`, `taplo` order** (SO Survey 2026 popularity order; a missing tool counts as `127` in its slot), or `0` only if every formatter succeeded.

The rule is deterministic regardless of completion order in the concurrent default.

For example:

- If `black` returns `2` and `shfmt` returns `5`, `all` exits `5` (`shfmt` registers after `black`).
- If `shfmt` returns `5` and the rest succeed, `all` exits `5`.

With a config file, the same rule applies in config-command order: the last non-zero code among the listed commands wins (a missing tool counts as `127` in its command slot and does not stop later commands), or `0` only if every command succeeds. Invalid or missing config files exit `2` before any formatter runs.

Use `--verbose` to see which executable was resolved when a tool is present.

`formattington --help`, `formattington --version`, and `formattington list` do not require any formatter executables and continue to work when tools are absent.

### Install hints

- `prettier` is installed via npm (`npm install -g prettier`).
- `sqlfluff` is the dialect-aware SQL linter/formatter; install with `pipx install sqlfluff` / `pip install sqlfluff`. There is no official binary and SQLFluff is Python-only, so it is PATH-only (a shared embedded Python runtime is tracked separately and was not introduced). Set the SQL dialect in the project config (`.sqlfluff` / `[tool.sqlfluff]` in `pyproject.toml`) or pass `--dialect` — Formattington never hardcodes one (SQLFluff falls back to `ansi` otherwise).
- `black` is installed via pip (`pip install black` or `uv tool install black`).
- `shfmt` comes from [mvdan/sh](https://github.com/mvdan/sh).
- `google-java-format` formats Java; the full release archives bundle the four upstream native binaries (macOS arm64, Linux arm64, Linux x86_64, Windows x86_64 — no macOS x86_64 or Windows arm64 asset exists upstream). Its non-whitespace `--cleanup` pass is disabled by default (`--cleanup=false`) so diffs stay formatting-only; pass `--cleanup=true` to opt in. For macOS x86_64 and Windows arm64 use the all-deps jar on a JDK ≥ 21 with the six `--add-exports` flags (`jdk.compiler/com.sun.tools.javac.{api,code,file,parser,tree,util}=ALL-UNNAMED`).
- `dotnet-format` is the .NET SDK's in-box formatter (≥ 6); it is PATH-only via the .NET SDK. Formattington invokes the `whitespace` subcommand in `--folder` mode with `--include <files>`, which needs **no MSBuild project or solution** and never runs project code; `--verify-no-changes` (check mode) exits 2, reported as 1. VB.NET is covered by the same binary (`.vb`). **Trust boundary:** the `style` and `analyzers` subcommands can restore and compile project code — they are available verbatim (`formattington dotnet-format style --severity info src/app.csproj`) but are never invoked by default. `.editorconfig` is the only configuration source.
- `powershell` lints with [PSScriptAnalyzer](https://github.com/microsoft/PSScriptAnalyzer) through PowerShell 7+ (`pwsh`); install the module with `Install-Module PSScriptAnalyzer` inside `pwsh` (Windows PowerShell 5.1, `powershell.exe`, is tried as a fallback).
- `clang-format` is part of LLVM ([documentation](https://clang.llvm.org/docs/ClangFormat.html)).
- `rustfmt` is the Rust formatter; install with `rustup component add rustfmt`. The edition defaults to rustfmt's own default (2015) — silently wrong for modern crates — so Formattington walks up from each file for the nearest Cargo.toml: `[package] edition` wins, and a member inheriting from a virtual workspace root uses the nearest `[workspace.package] edition`; files are grouped per edition before batching. Pass `--edition 2021` (etc.) verbatim to override detection or for repos with no Cargo.toml. `cargo fmt` reads the edition from Cargo.toml and is crate-aware, but it is not a file-list tool — `rustfmt` is used instead. `--check` exits 1 when files differ (also used for parse errors — indistinguishable; stderr disambiguates).
- `php-cs-fixer` is the de-facto PHP fixer; install with `composer global require friendsofphp/php-cs-fixer` or `brew install php-cs-fixer` (needs a PHP ≥ 7.4 interpreter on `PATH`). Its native check exit code is 8; Formattington reports it as 1.
- `gofmt` is part of the [Go](https://go.dev/) installation (install Go for your OS; `gofmt` comes with it).
- `goimports` formats like `gofmt` and also fixes import grouping; install with `go install golang.org/x/tools/cmd/goimports@latest`.
- `ktfmt` formats Kotlin; the full release archives bundle the three upstream native binaries (Linux x86_64, macOS arm64, Windows x86_64 — no Linux arm64, macOS x86_64, or Windows arm64 asset exists upstream; those stay PATH-only). No style flag is injected — pass `--google-style`/`--kotlinlang-style`/`--meta-style` verbatim as needed.
- `stylua` is the standard Lua formatter; a pinned copy is bundled in the full release archives, or install with `brew install stylua` / `cargo install stylua`.
- `swift-format` ships with Swift 6+ toolchains (Xcode 16+): it is PATH-only via `swift format` on macOS, or a standalone `swift-format` binary (brew, or any Swift toolchain build) tried first. Unparsable files are skipped (`--ignore-unparsable-files`). Configuration is `.swift-format` (JSON, discovered walking up from each file; `--configuration` overrides). `swift format lint --strict` exit 1 = findings (without `--strict` findings are warnings → exit 0); Formattington always uses `--strict` in check mode.
- `rubocop` is a **linter with ~100 layout cops**, integrated like powershell/markdownlint, not like a formatter: plain `formattington rubocop` lints (it never writes); pass `-x` for layout-only fixes (`formattington rubocop -x`) or `-a`/`-A` for all auto-corrections. Findings exit 1, config errors exit 2. Projects should pin their own version (a project Gemfile, or `gem install rubocop`); Ruby ≥ 2.7 required (3.x recommended). `.rubocop.yml` is hierarchical (`inherit_from`) and discovered natively. Alternatives: [Rufo](https://github.com/ruby-formatter/rufo) (dedicated formatter, `--check` exits 3) and [Standard](https://github.com/standardrb/standard) (opinionated RuboCop wrapper).
- `dart-format` is Dart's built-in formatter (`dart format`); install the Dart SDK for your platform. Configuration lives in `analysis_options.yaml` under a `formatter:` section (`page_width`, `trailing_commas: preserve`; language version ≥ 3.7) or per-file with `// dart format width=NNN` markers — discovered by the tool itself. `--set-exit-if-changed -o none` exits 1 when changes are needed and writes nothing (`-o none` = no output).
- `styler` is the de-facto R formatter (tidyverse style); it is an R package invoked through `Rscript -e`, so it is PATH-only and needs a working R installation (`apt install r-base` / `brew install r` / CRAN installer) plus `install.packages("styler")`. The check mode maps to `style_file(..., dry = "fail")`, which errors out (non-zero) when a file would be modified — reported as 1. **CRAN ships a newer release than the stale GitHub tags** (CRAN 1.11.0 as of late 2025; GitHub tags trail at v1.10.3 since 2024-04) — install from CRAN. User flags cannot be spliced into the generated R expression, so forwarded flags are a usage error; customize through a wrapper script or R-level `options()` (e.g. `styler::tidyverse_style()`).
- `nixfmt` is the official Nix formatter (RFC 166); install via Nix (`nixpkgs.nixfmt-rfc-style`) or `brew install nixfmt`.
- `gdformat` is the Godot 4 GDScript formatter (from the gdtoolkit package); install with `pipx install gdtoolkit==4.*` (3.x targets the Godot 3 line). It is PATH-only; Formattington does not set `--fast`, so the safety checks stay on.
- `npm-groovy-lint` is the only viable Groovy formatter (Spotless is a Gradle/Maven build plugin with no standalone CLI; Apache Groovy ships no formatter). Layout-only — it fixes indentation/brace spacing/blank lines and never re-breaks long lines. Requires **Node ≥ 18 and Java 11+** on `PATH` (a long-lived Java microservice backs the analysis; `--noserver` disables it). The `format` preset must be enabled in `.groovylintrc` (for example `extends: ["recommended", "format"]`) or layout findings are not reported. `Jenkinsfile` has no extension and is out of scope for discovery — pass it explicitly. Jenkins pipelines and Gradle `.gradle` files are covered (the `.gradle` extension is registered).
- `perltidy` is the Perl formatter; install from CPAN (`cpan Perl::Tidy`) or your package manager. It is GPL-2.0+ and is never bundled into the release binaries (and requires `perl` on `PATH` — on Windows, install Strawberry Perl).
- `zig-fmt` is part of the Zig toolchain (`zig fmt`); install [Zig](https://ziglang.org/download/) for your platform. **Zig is pre-1.0: formatting can drift between toolchain versions** — Formattington always invokes the zig found on `PATH`, which matches the project's toolchain by construction. Zero configuration by design — there is no config file and none is invented. `.zon` files are NOT registered; pass `--zon` verbatim to include them. `zig fmt --check` exits 1 when any file would change (missing files also exit 1; the stderr diagnostic disambiguates).
- `mix-format` is Elixir's built-in formatter (`mix format`); install [Elixir](https://elixir-lang.org/install.html) for your platform — it ships `mix` and needs Erlang/OTP. **`.formatter.exs` is EXECUTED CODE** (returns a keyword list of formatter inputs/plugins), discovered from the CURRENT DIRECTORY — never run it against untrusted projects. Formatter output can change between Elixir versions. `--no-compile` is passed by default so formatting never compiles; drop it (or set formatter plugins) when plugins are needed. `mix format --check-formatted --no-compile` exits non-zero when formatting is needed (reported as 1).
- `fourmolu` formats Haskell; the full release archives bundle the four upstream binaries (macOS arm64, macOS x86_64, Linux arm64, Linux x86_64 — no Windows asset upstream, so Windows stays PATH-only via `ghcup install fourmolu` or `cabal install fourmolu`). Linux builds are glibc-dynamic (musl/Alpine users need the PATH route). Its `--mode check` prints a unified diff to **stderr** and exits **100** when files need formatting (0 clean; 102/101 on other failures — passed through verbatim). Configuration is `fourmolu.yaml` (walks up from each file; `--config` forwarded verbatim); literate Haskell (`.lhs`) is not registered (unverified against the released parser).
- `scalafmt` formats Scala; it is PATH-only via coursier (`cs install scalafmt`) or brew (`brew install scalafmt`). **Why PATH-only:** scalafmt requires a `version = <x.y.z>` line in `.scalafmt.conf`, and the upstream native release binaries have no dynamic runner — a config whose pinned version differs from the compiled-in one fails hard (exact error: `error: invalid Scalafmt version.`, exit bit 16). The coursier/JVM launcher honors the pin by downloading it dynamically, so PATH-only is the only route with full fidelity. The `.sc` worksheet extension is covered.
- `fprettify` is the de-facto Fortran formatter; install with `pipx install fprettify` / `pip install fprettify`. It is GPL-3.0+ and Python-based, so it was deliberately rejected for bundling; free-form Fortran only (`.f90`, `.f95`, `.f03`, `.f08`, `.f18`). Its `--diff` does not affect the exit code, so Formattington synthesizes check mode from non-empty diff output.
- `erlfmt` is the mature Erlang formatter; there are no bundleable binaries (it ships as a rebar3 plugin or escript), so it is PATH-only and needs a working Erlang/OTP 21+ install. Invoke the escript form (`erlfmt -w` / `erlfmt -c`).
- `jlfmt` is the Julia formatter (JuliaFormatter.jl); it ships only as a Julia app (`julia -e 'using Pkg; Pkg.Apps.add("JuliaFormatter")'`, needs Julia ≥ 1.12 on `PATH`). There is no bundle — a ~600 MB runtime payload was never on the table; Windows on ARM has no Julia build at all. Configuration is `.JuliaFormatter.toml` (walks up from each file). `jlfmt --diff` is computed via `git --no-index` and thus needs git; without git on `PATH` Formattington prints a note on stderr and runs `--check` instead.
- `cljfmt` formats Clojure; the full release archives bundle the four upstream native binaries (macOS arm64, Linux arm64, Linux x86_64 static, Windows x86_64 — no macOS x86_64 or Windows arm64 asset upstream). cljfmt is distributed under **EPL-1.0**; the redistributed binaries are unmodified upstream build artifacts (attribution in this README). `cljfmt check` exits 1 when files need formatting (2 on parse/format error, passed through verbatim); configuration is `.cljfmt.edn`/`cljfmt.edn` discovered by the tool itself.
- `gleam-format` is part of the Gleam toolchain (`gleam format`); install [Gleam](https://gleam.run/getting-started/installing-gleam/) for your platform. No configuration by design (fixed style). `gleam format --check` exits non-zero (Error::Format, i.e. 1) when files would change — reported as 1.
- `gnatformat` formats Ada (AdaCore/GNATformat); it ships with **GNAT Pro** (commercial) or via **Alire** (`alr install gnatformat`, binary crate `gnatformat_bin`) — there is no standalone release binary to bundle. **Project-file model:** options come from the `.gpr` file; when exactly one `.gpr` exists in the cwd the tool auto-loads it, and when none exists Formattington passes `--no-project` (standalone defaults). `.ads`/`.adb` are covered; the `.gpr` file itself is configuration, not formatted. `--check` exits 1 when files are not formatted; `--pipe` prints the formatted source for your own `diff`.
- `fantomas` is the F# formatter; it ships only as a .NET global tool (no native binaries, no runtime to embed), so it is PATH-only. Install with `dotnet tool install -g fantomas` (needs the .NET SDK ≥ 8). Its `--check` exits **99** when formatting is needed (1 on error); Formattington reports 99 as 1 and passes 1 through. Configuration is `.editorconfig` (plus `.fantomasignore`), discovered by the tool itself.
- `ocamlformat` formats OCaml; it ships only as opam source (built from source against your switch), so it is PATH-only — there is no bundle. Install with `opam install ocamlformat`. **The `.ocamlformat` file requires a `version = <installed release>` line** and the tool refuses to run on a mismatch — a common failure mode; pin `version =` to the release you have installed. `--check` exit 1 means formatting needed *or* error (indistinguishable — stderr disambiguates). `.ocamlformat-ignore` exists. `dune fmt` wraps it for dune projects.
- `forge-fmt` is the standard Solidity formatter, from Foundry (`forge fmt`); install via `foundryup`, Homebrew (`brew install foundry`), or release tarballs from github.com/foundry-rs/foundry/releases. `forge fmt --check` exits 1 when formatting is required and prints a diff (`-r` for raw output). Configuration is the `[fmt]` table in `foundry.toml` (discovered from the project root/git root; `--nearest` uses each file's nearest `foundry.toml`). No Windows ARM64 build — use the x86_64 binary under emulation or WSL.
- `mojo-format` is part of the Mojo toolchain (`mojo format`); install from Modular (`uv pip install mojo` / `pixi add mojo`). **Platform support is narrow: Linux glibc ≥ 2.34 (x86-64-v3 / Neoverse N1+) and macOS Apple silicon only; Windows requires WSL.** **There is no check mode upstream** — Formattington synthesizes one by copying each file to a temp dir, formatting the copies, and byte-comparing (the user tree is never modified); tool errors pass through verbatim. `-l/--line-length` is forwarded verbatim. No configuration file exists and none is invented.
- `odinfmt` formats Odin; the only implementation lives in the Odin Language Server (`DanielGavin/ols`, community-maintained, not officially blessed by odin-lang — no release binaries; no Windows binary at all). Install with `brew install odinfmt` or build from ols source; `odinfmt.json` is discovered by walking up from the target (`--config` overrides). **`-w` creates an `<file>_bk` backup** before replacing the file — add `*_bk` (sic) to your ignore rules. There is **no upstream check mode**; check mode is synthesized by byte-comparing `odinfmt <file>` stdout against the original (tree untouched; one process per file).
- `elm-format` formats Elm; the full release archives bundle the five upstream static binaries (Linux x86_64/arm64, macOS x86_64/arm64, Windows x86_64 — no Windows arm64 asset upstream; that platform stays PATH-only). Zero-configuration by design — there is no config file. The mac-arm64 asset is ~42 MB extracted (an unstripped upstream build, redistributed unmodified). `--validate` prints a JSON report and exits 1 when files need formatting (0 clean; both modes are error-ambiguous but the report disambiguates).
- `markdownlint` (`markdownlint-cli`) lints Markdown files (it never rewrites without an explicit `--fix`); the full release archives bundle it, or install with `npm install -g markdownlint-cli`. Configuration (for example `.markdownlint.json`) is discovered natively; findings exit 1.
- `taplo` (`taplo fmt`) formats TOML; a pinned copy is bundled in the full release archives under `formatters/taplo/<version>/<platform>/` for offline use (download the release binary and put it on `PATH` if you prefer a system install, or fetch one into your checkout with `formatters/taplo/update.sh`).
Either a bundled copy under `formatters/` (see [Bundled tools](#bundled-tools)) — or, in the full release builds, the copy embedded inside the binary — or an installation on `PATH` satisfies `FindTool()`. `gofmt` and `goimports` are PATH-only and are found via `PATH`.

## Supported tools

Formattington integrates existing tools rather than reimplementing their functionality. Tools are presented in Stack Overflow Developer Survey 2026 popularity order — see [Presentation order in AGENTS.md](AGENTS.md#presentation-order).

| Tool                                                              | Category  | Status    | Typical use                                             |
| ----------------------------------------------------------------- | --------- | --------- | ------------------------------------------------------- |
| [Prettier](https://prettier.io/)                                  | Formatter | Available | JavaScript, TypeScript, JSON, HTML, CSS, Markdown, YAML |
| [SQLFluff](https://github.com/sqlfluff/sqlfluff)                          | Linter/Formatter | Available | SQL (`.sql`) via `sqlfluff fix` — PATH-only                 |
| [Black](https://black.readthedocs.io/)                            | Formatter | Available | Python (`.py`)                                          |
| [shfmt](https://github.com/mvdan/sh)                              | Formatter | Available | Shell scripts                                           |
| [google-java-format](https://github.com/google/google-java-format)        | Formatter | Available | Java (`.java`)                                               |
| [dotnet format](https://learn.microsoft.com/dotnet/core/tools/dotnet-format) | Formatter | Available | C#/VB.NET (`.cs`, `.vb`) — PATH-only; .NET SDK ≥ 6          |
| [PSScriptAnalyzer](https://github.com/microsoft/PSScriptAnalyzer) | Linter    | Available | PowerShell scripts (`.ps1`, `.psm1`) via `pwsh`         |
| [clang-format](https://clang.llvm.org/docs/ClangFormat.html)      | Formatter | Available | C, C++, and related languages                           |
| [rustfmt](https://github.com/rust-lang/rustfmt)                             | Formatter | Available | Rust (`.rs`) — PATH-only; rustup component                 |
| [PHP-CS-Fixer](https://github.com/PHP-CS-Fixer/PHP-CS-Fixer)                       | Formatter | Available | PHP (`.php`) via `php-cs-fixer fix`                     |
| [gofmt](https://pkg.go.dev/cmd/gofmt)                             | Formatter | Available | Go (`.go`)                                              |
| [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)  | Formatter | Available | Go (`.go`) with import fixing                           |
| [ktfmt](https://github.com/Kotlin/ktfmt)                                 | Formatter | Available | Kotlin (`.kt`, `.kts`)                                       |
| [StyLua](https://github.com/JohnnyMorganz/StyLua)                                 | Formatter | Available | Lua (`.lua`)                                            |
| [swift-format](https://github.com/swiftlang/swift-format)                   | Formatter | Available | Swift (`.swift`) — PATH-only; Swift 6+ toolchain           |
| [RuboCop](https://github.com/rubocop/rubocop)                           | Linter    | Available | Ruby (`.rb`, `.rake`, `.gemspec`, `.ru`) — PATH-only          |
| [Dart SDK](https://dart.dev/get-dart)                                    | Formatter | Available | Dart (`.dart`) via `dart format` — PATH-only; Dart SDK    |
| [styler](https://github.com/r-lib/styler)                              | Formatter | Available | R (`.R`, `.r`) — PATH-only; needs Rscript                |
| [nixfmt](https://github.com/NixOS/nixfmt)                                         | Formatter | Available | Nix (`.nix`)                                            |
| [gdformat](https://github.com/Scony/godot-gdscript-toolkit)              | Formatter | Available | GDScript (`.gd`) — PATH-only, Godot 4                       |
| [npm-groovy-lint](https://github.com/nvuillam/npm-groovy-lint)          | Formatter | Available | Groovy (`.groovy`, `.gradle`) — PATH-only; Node + Java      |
| [perltidy](https://github.com/perltidy/perltidy)                              | Formatter | Available | Perl (`.pl`, `.pm`, `.t`) — PATH-only, GPL-2.0+            |
| [zig fmt](https://ziglang.org/documentation/)                               | Formatter | Available | Zig (`.zig`) via `zig fmt` — PATH-only; Zig toolchain    |
| [mix format](https://hexdocs.pm/mix/Mix.Tasks.Format.html)                | Formatter | Available | Elixir (`.ex`, `.exs`) via `mix format` — PATH-only        |
| [Fourmolu](https://github.com/fourmolu/fourmolu)                          | Formatter | Available | Haskell (`.hs`)                                              |
| [scalafmt](https://github.com/scalameta/scalafmt)                         | Formatter | Available | Scala (`.scala`, `.sbt`, `.sc`) — PATH-only                 |
| [fprettify](https://github.com/pseewald/fprettify)                           | Formatter | Available | Fortran free-form (`.f90`…) — PATH-only, GPL-3.0+           |
| [erlfmt](https://github.com/WhatsApp/erlfmt)                                | Formatter | Available | Erlang (`.erl`, `.hrl`) — PATH-only; needs OTP 21+          |
| [JuliaFormatter.jl](https://github.com/JuliaEditorSupport/JuliaFormatter.jl)   | Formatter | Available | Julia (`.jl`) via `jlfmt` — PATH-only                   |
| [cljfmt](https://github.com/weavejester/cljfmt)                          | Formatter | Available | Clojure (`.clj`, `.cljs`, `.cljc`, `.edn`)                   |
| [gleam format](https://gleam.run/book/introduction/formatting.html)          | Formatter | Available | Gleam (`.gleam`) via `gleam format` — PATH-only          |
| [GNATformat](https://github.com/AdaCore/GNATformat)                     | Formatter | Available | Ada (`.ads`, `.adb`) — PATH-only; GNAT Pro / Alire          |
| [Fantomas](https://github.com/fsprojects/fantomas)                         | Formatter | Available | F# (`.fs`, `.fsi`, `.fsx`) — PATH-only; .NET SDK required   |
| [ocamlformat](https://github.com/ocaml-ppx/ocamlformat)                    | Formatter | Available | OCaml (`.ml`, `.mli`) — PATH-only; needs opam                |
| [forge fmt](https://github.com/foundry-rs/foundry) (Foundry)           | Formatter | Available | Solidity (`.sol`) via `forge fmt` — PATH-only               |
| [mojo format](https://docs.modular.com/mojo/manual/format/)                 | Formatter | Available | Mojo (`.mojo`) — Linux/macOS arm64 or WSL — PATH-only      |
| [odinfmt](https://github.com/DanielGavin/ols) (community)               | Formatter | Available | Odin (`.odin`) — PATH-only; macOS/Linux only, no Windows   |
| [elm-format](https://github.com/avh4/elm-format)                          | Formatter | Available | Elm (`.elm`)                                              |
| [markdownlint](https://github.com/DavidAnson/markdownlint)        | Linter    | Available | Markdown style and consistency via `markdownlint-cli`   |
| [Taplo](https://taplo.tamasfe.dev/)                               | Formatter | Available | TOML (`.toml`) via `taplo fmt`                          |

Only `prettier`, `sqlfluff`, `black`, `shfmt`, `google-java-format`, `dotnet-format`, `powershell`, `clang-format`, `rustfmt`, `php-cs-fixer`, `gofmt`, `goimports`, `ktfmt`, `stylua`, `swift-format`, `rubocop`, `dart-format`, `styler`, `nixfmt`, `gdformat`, `npm-groovy-lint`, `perltidy`, `zig-fmt`, `mix-format`, `fourmolu`, `scalafmt`, `fprettify`, `erlfmt`, `jlfmt`, `cljfmt`, `gleam-format`, `gnatformat`, `fantomas`, `ocamlformat`, `forge-fmt`, `mojo-format`, `odinfmt`, `elm-format`, `markdownlint`, `taplo`, `md`/`markdown`/`html` (aliases for `prettier`), `sql` (alias for `sqlfluff`), `python`/`py`/`micropython` (aliases for `black`), `bash` (alias for `shfmt`), `java` (alias for `google-java-format`), `csharp`/`vbnet` (aliases for `dotnet-format`), `ps1` (alias for `powershell`), `c`/`cpp`/`c++` (aliases for `clang-format`), `rust` (alias for `rustfmt`), `php` (alias for `php-cs-fixer`), `go` (alias for `gofmt`), `kotlin` (alias for `ktfmt`), `lua` (alias for `stylua`), `swift` (alias for `swift-format`), `ruby` (alias for `rubocop`), `dart` (alias for `dart-format`), `r` (alias for `styler`), `nix` (alias for `nixfmt`), `gdscript` (alias for `gdformat`), `groovy` (alias for `npm-groovy-lint`), `perl` (alias for `perltidy`), `zig` (alias for `zig-fmt`), `elixir` (alias for `mix-format`), `haskell` (alias for `fourmolu`), `scala` (alias for `scalafmt`), `fortran` (alias for `fprettify`), `erlang` (alias for `erlfmt`), `julia` (alias for `jlfmt`), `clojure` (alias for `cljfmt`), `gleam` (alias for `gleam-format`), `ada` (alias for `gnatformat`), `fsharp` (alias for `fantomas`), `ocaml` (alias for `ocamlformat`), `solidity` (alias for `forge-fmt`), `mojo` (alias for `mojo-format`), `odin` (alias for `odinfmt`), `elm` (alias for `elm-format`), `toml` (alias for `taplo`), and `all` are accepted by the CLI.

Prettier is the sole owner of Markdown formatting. There is no dedicated Markdown formatter module; `formattington md` and `formattington markdown` resolve to the existing Prettier integration, which covers `.md` and `.markdown` files and respects Prettier's native configuration (such as `.prettierrc`). Markdown _style_ is a distinct operation and is linted by `formattington markdownlint`: it runs [`markdownlint-cli`](https://github.com/igorshubovych/markdownlint-cli) against `.md`/`.markdown` files, discovery batches files as usual, findings exit `1`, and no flags are injected — the tool is read-only without an explicit `--fix`, and `--fix`, `--config`, and glob patterns are forwarded verbatim, so it edits files only when you ask. Configuration is discovered natively by the tool (for example `.markdownlint.json` in the working directory) and `.prettierrc`'s sibling `.markdownlint.json` needs no Formattington-side setup. Aliases are not involved: use `md`/`markdown` for formatting and `markdownlint` for linting. Unrecognized names still report a usage error (exit 2).

Go has two formatters covering the same `.go` files: `gofmt` (canonical formatting) and `goimports` (formatting plus import grouping). `formattington go` resolves to `gofmt`; use `formattington goimports` or a config command listing `goimports` when imports should also be fixed. Use `--series` with `all` to run them sequentially (`gofmt` then `goimports`) instead of concurrently, or list only one in a config file to avoid running both against the same files. Per AGENTS.md, when multiple tools support the same file type, selection should respect configured tool choices.

MicroPython is covered by the Python formatter: MicroPython is a Python 3 subset, so `formattington micropython` (alias for `black`) and `formattington python` both format `.py` MicroPython files — fixtures using `micropython` imports, `@micropython.native`/`@micropython.viper`, and inline-assembly calls in `@micropython.asm_viper`/`@micropython.asm_thumb` parse and format cleanly under black 26.5.1 and `ruff format` (the two agree on the output). The caveat: constructs that are not valid Python — notably statement-level `label:` statements inside inline-assembly functions — cannot be parsed by black (or ruff; `Cannot parse: <line>:<col>`), so those functions must be excluded or kept minimal by hand.

Additional tools may provide other capabilities, such as checking, linting, and fixing, without necessarily being formatters.

### Prerequisites

| Tool         | Install method (bundled)                                                                                                                                                | Install method (PATH)                                                                                        | How Formattington finds it                                              |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| prettier     | Fetched by `formatters/prettier/update.sh`; bundled in the full release archives                                                                                        | `npm install -g prettier`                                                                                    | Bundled (or embedded) first, then `PATH` via `FindTool("prettier")`     |
| sqlfluff     | Not bundled (PATH-only; Python-only; no official binaries)                                                                                                             | `pipx install sqlfluff` / `pip install sqlfluff`                                                              | `PATH` via `FindTool("sqlfluff")`                                   |
| black        | Fetched by `formatters/black/update.sh` (its macOS x86_64 build is compiled from source by the same script on a native Intel Mac); bundled in the full release archives | `pip install black` or `uv tool install black`                                                               | Bundled (or embedded) first, then `PATH` via `FindTool("black")`        |
| shfmt        | Fetched by `formatters/shfmt/update.sh`; bundled in the full release archives                                                                                           | `go install mvdan.cc/sh/v3/cmd/shfmt@latest` or your OS package manager                                      | Bundled (or embedded) first, then `PATH` via `FindTool("shfmt")`        |
| google-java-format | Fetched by `formatters/google-java-format/update.sh`; bundled for macOS arm64, Linux arm64/x86_64, Windows x86_64 | Download the release archive, or `brew install google-java-format`                                     | Bundled (or embedded) first, then `PATH` via `FindTool("google-java-format")` |
| dotnet-format | Not bundled (PATH-only; ships in-box with the .NET SDK ≥ 6)                                                                                                            | [.NET SDK](https://dot.net/download) for your OS                                                     | `PATH` via `FindTool("dotnet")`                                   |
| powershell   | Not bundled (PATH-only; PSScriptAnalyzer is a module)                                                                                                                   | PowerShell 7+ (`pwsh`) on `PATH` plus `Install-Module PSScriptAnalyzer` in `pwsh`                            | `PATH` via `FindTool("pwsh")`, then `FindTool("powershell")`            |
| clang-format | Fetched by `formatters/clang-format/update.sh`; bundled in the full release archives                                                                                    | Install LLVM/clang-tools for your OS (for example `brew install clang-format` or `apt install clang-format`) | Bundled (or embedded) first, then `PATH` via `FindTool("clang-format")` |
| rustfmt      | Not bundled (PATH-only; rustup component)                                                                                                                                  | `rustup component add rustfmt`                                                                            | `PATH` via `FindTool("rustfmt")`                                    |
| php-cs-fixer | Not bundled (PATH-only; needs a PHP interpreter)                                                                                                                          | `composer global require friendsofphp/php-cs-fixer` / `brew install php-cs-fixer` (PHP ≥ 7.4 on `PATH`)        | `PATH` via `FindTool("php-cs-fixer")`                               |
| gofmt        | Not bundled (PATH-only)                                                                                                                                                 | Install Go for your OS; `gofmt` comes with the Go installation                                               | `PATH` via `FindTool("gofmt")`                                          |
| goimports    | Not bundled (PATH-only)                                                                                                                                                 | `go install golang.org/x/tools/cmd/goimports@latest` (ensure `$(go env GOPATH)/bin` is on `PATH`)            | `PATH` via `FindTool("goimports")`                                      |
| ktfmt        | Fetched by `formatters/ktfmt/update.sh`; bundled for Linux x86_64, macOS arm64, Windows x86_64                        | `brew install ktfmt` or a PATH install of ktfmt (or the with-dependencies jar, JDK 11+)                    | Bundled (or embedded) first, then `PATH` via `FindTool("ktfmt")`           |
| stylua       | Fetched by `formatters/stylua/update.sh`; bundled in the full release archives (no Windows arm64 asset upstream)                                                          | `brew install stylua` or `cargo install stylua`                                                              | Bundled (or embedded) first, then `PATH` via `FindTool("stylua")`     |
| swift-format | Not bundled (PATH-only; ships with Swift 6+ toolchains)                                                                                                                        | Xcode 16+ / Swift 6+ toolchain; or `brew install swift-format`                                       | standalone `swift-format` on `PATH`, else `swift format`                                |
| rubocop      | Not bundled (PATH-only; gem-only; projects pin their own version)                                                                                                      | `gem install rubocop` / project Gemfile (Ruby ≥ 2.7)                                                  | `PATH` via `FindTool("rubocop")`                                    |
| dart-format  | Not bundled (PATH-only; ships with the Dart SDK)                                                                                                                            | `brew install dart` / Dart SDK installer                                                          | `PATH` via `FindTool("dart")`                                       |
| styler       | Not bundled (PATH-only; R package; needs `Rscript` on PATH + `install.packages("styler")`)                                                                                | `install.packages("styler")` in R                                                               | `PATH` via `FindTool("Rscript")`                                     |
| nixfmt       | Not bundled (PATH-only; upstream publishes only a static Linux x86_64 asset)                                                                                             | `nixpkgs.nixfmt-rfc-style` / `brew install nixfmt`                                                           | `PATH` via `FindTool("nixfmt")`                                      |
| gdformat     | Not bundled (PATH-only; Python package; no official binaries)                                                                                                               | `pipx install gdtoolkit==4.*`                                                                      | `PATH` via `FindTool("gdformat")`                                    |
| npm-groovy-lint | Not bundled (PATH-only; needs Node ≥ 18 and Java 11+)                                                                                                                | `npm install -g npm-groovy-lint`                                                                          | `PATH` via `FindTool("npm-groovy-lint")`                               |
| perltidy     | Not bundled (PATH-only; GPL-2.0+; needs `perl` on `PATH`)                                                                                                                | `cpan Perl::Tidy` / OS package manager (Windows: Strawberry Perl)                                              | `PATH` via `FindTool("perltidy")`                                  |
| zig-fmt      | Not bundled (PATH-only; ships with the Zig toolchain)                                                                                                                    | `brew install zig` / [ziglang.org](https://ziglang.org/download/)                                  | `PATH` via `FindTool("zig")`                                        |
| mix-format   | Not bundled (PATH-only; ships with Elixir)                                                                                                                                | [Elixir installer](https://elixir-lang.org/install.html) for your OS (ships `mix`; needs Erlang/OTP)   | `PATH` via `FindTool("mix")`                                        |
| fourmolu     | Fetched by `formatters/fourmolu/update.sh`; bundled for macOS arm64/x86_64, Linux arm64/x86_64                              | `ghcup install fourmolu` / `cabal install fourmolu`                                               | Bundled (or embedded) first, then `PATH` via `FindTool("fourmolu")`             |
| scalafmt     | Not bundled (PATH-only; the natives fail hard on the `.scalafmt.conf` `version =` pin — verified: exit bit 16)              | `cs install scalafmt` / `brew install scalafmt` (JVM; honors the pin dynamically)            | `PATH` via `FindTool("scalafmt")`                                    |
| fprettify    | Not bundled (PATH-only; GPL-3.0+; needs Python 3 on `PATH`)                                                                                                               | `pipx install fprettify` / `pip install fprettify`                                                             | `PATH` via `FindTool("fprettify")`                                  |
| erlfmt       | Not bundled (PATH-only; no bundleable binaries; needs Erlang/OTP 21+)                                                                                              | `brew install erlang` / OTP package of choice; escript built from rebar3 (`rebar3 as release escriptize`)   | `PATH` via `FindTool("erlfmt")`                                     |
| jlfmt        | Not bundled (PATH-only; Julia app; ~600 MB runtime; no Windows ARM Julia build)                                                                                               | `julia -e 'using Pkg; Pkg.Apps.add("JuliaFormatter")'` (Julia ≥ 1.12)                                  | `PATH` via `FindTool("jlfmt")`                                      |
| cljfmt       | Fetched by `formatters/cljfmt/update.sh`; bundled for macOS arm64, Linux arm64/x86_64-static, Windows x86_64 (EPL-1.0) | Download the release archive, or `brew install cljfmt`                                            | Bundled (or embedded) first, then `PATH` via `FindTool("cljfmt")`             |
| gleam-format | Not bundled (PATH-only; ships with the Gleam toolchain)                                                                                                                     | [Gleam installer](https://gleam.run/getting-started/installing-gleam/) for your OS                     | `PATH` via `FindTool("gleam")`                                      |
| gnatformat   | Not bundled (PATH-only; ships with GNAT Pro or Alire)                                                                                                                       | GNAT Pro, or `alr install gnatformat` (Alire)                                                            | `PATH` via `FindTool("gnatformat")`                                  |
| fantomas     | Not bundled (PATH-only; ships only as a .NET global tool)                                                                                                                   | `dotnet tool install -g fantomas` (.NET SDK ≥ 8)                                                      | `PATH` via `FindTool("fantomas")`                                   |
| ocamlformat  | Not bundled (PATH-only; opam source build; no native binaries exist)                                                                                                      | `opam install ocamlformat` (requires a working opam switch)                                           | `PATH` via `FindTool("ocamlformat")`                                 |
| forge-fmt    | Not bundled (PATH-only; ships with Foundry)                                                                                                                                | `foundryup` / `brew install foundry` / npm                                                         | `PATH` via `FindTool("forge")`                                       |
| mojo-format  | Not bundled (PATH-only; Linux glibc ≥ 2.34 / macOS arm64 only; Windows via WSL)                                                                                              | Modular channel: `uv pip install mojo` or `pixi add mojo`                                           | `PATH` via `FindTool("mojo")`                                       |
| odinfmt      | Not bundled (PATH-only; community tool; no release binaries; no Windows binary)                                                                                               | `brew install odinfmt` / build from `DanielGavin/ols` source                                      | `PATH` via `FindTool("odinfmt")`                                    |
| elm-format   | Fetched by `formatters/elm-format/update.sh`; bundled for Linux x86_64/arm64, macOS x86_64/arm64, Windows x86_64 | `npm install -g elm-format` / `brew install elm-format`                                   | Bundled (or embedded) first, then `PATH` via `FindTool("elm-format")`        |
| markdownlint | Fetched by `formatters/markdownlint/update.sh`; bundled in the full release archives                                                                        | `npm install -g markdownlint-cli`                                                                            | Bundled (or embedded) first, then `PATH` via `FindTool("markdownlint")` |
| taplo        | Fetched by `formatters/taplo/update.sh`; bundled in the full release archives                                                                                           | Download taplo from https://github.com/tamasfe/taplo releases and put `taplo` on `PATH`                      | Bundled (or embedded) first, then `PATH` via `FindTool("taplo")`        |

Bundled versions are pinned by the `formatters/<tool>/update.sh` scripts for offline use in the release archives; see [Bundled tools](#bundled-tools) for the layout and refresh process.

If neither a bundle nor a `PATH` installation is found, the CLI prints `formattington: Could not find ...` to stderr and exits 127 (see [Exit codes](#exit-codes)).

Formatter configuration files such as `.prettierrc` and `.clang-format` remain in your repository and are respected by the underlying tools.

## Installation

Formattington is a Go program. It has no third-party dependencies, so `go build` is all that is needed.

Clone the repository:

```bash
git clone https://github.com/fordaj/formattington.git
cd formattington
```

Build the binary:

```bash
make build
```

`make build` writes the executable to `./bin/formattington`. A plain `go build -o bin/formattington ./cmd/formattington` works just as well.

The project requires Go 1.24 or newer (`go.mod` declares the language version).

Run Formattington from the built binary:

```bash
./bin/formattington all

./bin/formattington --version
```

To use it from anywhere, put the binary on your `PATH`:

```bash
go install github.com/fordaj/formattington/cmd/formattington@latest
```

### Tool dependencies

Formattington orchestrates external tools; it does not automatically make every tool available simply by building the program.

A fresh clone contains no formatter binaries (they are not committed); fetch them into your checkout with `formatters/<tool>/update.sh`, or download a `formattington-<platform>-full` release archive, which is a single self-contained binary with the formatters embedded.

If you prefer a system installation, put the tool on `PATH` instead — `FindTool()` prefers bundled copies and falls back to `PATH`.

`gofmt` (part of the Go installation) and `goimports` (installed via `go install golang.org/x/tools/cmd/goimports@latest`) are PATH-only and are not bundled.

See the [prerequisites table](#prerequisites) for per-tool options.

## How it works

Formattington separates the orchestration layer from the tools themselves.

1. **Discover** matching files in the project.
2. **Select** tools based on the requested language or tool name.
3. **Execute** the corresponding tools with the appropriate arguments (concurrently by default with `all`; one after another with `all --series`).
4. **Report** the results through a common interface (last non-zero exit code in tool order, `0` only if all succeed).

Formattington owns orchestration, file discovery, tool selection, and execution strategy. Individual tools retain responsibility for their specialized functionality, native configuration, and rules.

The goal is to keep the interface simple while letting established tools do what they do best.

## Bundled tools

Formattington looks for bundled tool executables under:

```text
formatters/<name>/v<version>/<platform>/
```

Here, `<platform>` is the hyphenated `<os>-<arch>` name returned by `PlatformDir()` (for example `linux-x86_64`, `macos-arm64`, `windows-x86_64`). The bundled executables are not committed to the repository; `release.bash` fetches them at release time by running `formatters/<name>/update.sh`, and that directory (or a `FORMATTINGTON_FORMATTERS_DIR` override) is the lookup root.

The release `formattington-<platform>-full` binaries embed their platform's `formatters/` tree as a zip inside the executable (Go `full` build tag + `go:embed`). On first use the tree is unpacked into the per-user cache directory (`formatters/` under `os.UserCacheDir()`, for example `~/Library/Caches/formattington/formatters` on macOS) and reused from then on; upgrading to a binary with different embedded formatters replaces the extraction wholesale. The `formattington-<platform>-lite` binaries embed nothing.

Bundled tools are preferred; when none is found, lookup falls back to `PATH`. `formattington list` reports which bundled and PATH versions Formattington sees.

`gofmt` and `goimports` are PATH-only and are not bundled; install `gofmt` via the Go installation, and `goimports` via `go install golang.org/x/tools/cmd/goimports@latest`, placing the executables on `PATH`.

`markdownlint` is bundled in the full release archives, wired into the CLI (registered in survey order), and included in `all` like every other registered tool. `stylua` is bundled the same way (registered in survey order; no Windows arm64 asset is published upstream, so that platform relies on `PATH`).

Use `--verbose` before the tool name to see which executable was resolved, or set `FORMATTINGTON_VERBOSE=1`:

```bash
formattington --verbose shfmt
formattington --verbose --check shfmt
```

### Migration notes

The shfmt Linux bundles were previously stored under underscored names (`linux_arm64`, `linux_x86_64`). They are now hyphenated (`linux-arm64`, `linux-x86_64`) to match `PlatformDir()` and clang-format.

Underscored layouts (for example `linux_x86_64`) are no longer resolved; only the hyphenated layout is supported.

Bundled versions are compared numerically, so `v10` wins over `v9` and `v23.1.3` wins over `v22.1.8`.

On Windows, `PATH` lookup tries both `name` and `name.exe`.

### Repository hygiene

`bin/`, `dist/`, `*.exe`, `*.test`, `*.out`, `.DS_Store`, and `.logs/` are ignored via `.gitignore` and must not be tracked; so is `internal/formattington/assets/`, the staging directory `release.bash` fills with the embedded formatter payload for the full build.

The formatter executables are fetched at release time and are not committed; `tools/` carries only the pinned helper tools (versioner, logamuffin, runner-spawner).

## Testing

Run the test suite with:

```bash
make test
```

That is a thin wrapper around `go test ./...`. Tests live next to the code in `internal/formattington/` and cover CLI parsing, tool discovery, subprocess boundaries, argument forwarding, and the configuration file. They run without network access and without formatter binaries installed, using fake executable scripts in temporary directories.

`make check` additionally runs `go vet ./...` and verifies that `gofmt` has nothing to change.

## Releases

`release.bash` computes the next version with [versioner](https://github.com/fordaj/Versioner) (the bundled binary, selected via `scripts/entry.sh`), fetches the bundled formatter executables with `formatters/<tool>/update.sh`, cross-compiles the program for every supported platform, smoke-tests the freshly built full binary on the host, tags the commit, pushes the tag, and creates a GitHub release with the per-platform archives attached.

```bash
./release.bash                # version, fetch, build, tag, push, release + upload
./release.bash --build-only   # fetch, cross-compile into dist/, smoke-test, package; stop
./release.bash --dry-run      # compute + verify + build, touch nothing remote
```

Two archives are produced per platform — `formattington-<platform>-full` and `formattington-<platform>-lite`. The **full** binary embeds the formatter executables for its target platform inside itself (Go `full` build tag + `go:embed`, as a zip unpacked into the per-user cache on first use), so it is a single self-contained executable; the **lite** binary is just `formattington`. All formatter integrations resolve their bundled copies through `FormattersDir()` at runtime: the `FORMATTINGTON_FORMATTERS_DIR` environment variable wins, then the embedded formatters (full builds), then a `formatters/` directory next to the executable or in an ancestor directory (a repository checkout), then `PATH`.

The cross-compile matrix (CGO disabled, so the binaries are statically linked and portable):

| Platform        | `GOOS`/`GOARCH`   | Binary in `dist/`                                  | Release asset                                  |
| --------------- | ----------------- | -------------------------------------------------- | ---------------------------------------------- |
| `linux-x64`     | `linux`/`amd64`   | `dist/{lite,full}/linux-x64/formattington`         | `formattington-linux-x64-{lite,full}.tar.gz`   |
| `linux-arm64`   | `linux`/`arm64`   | `dist/{lite,full}/linux-arm64/formattington`       | `formattington-linux-arm64-{lite,full}.tar.gz` |
| `macos-x64`     | `darwin`/`amd64`  | `dist/{lite,full}/macos-x64/formattington`         | `formattington-macos-x64-{lite,full}.tar.gz`   |
| `macos-arm64`   | `darwin`/`arm64`  | `dist/{lite,full}/macos-arm64/formattington`       | `formattington-macos-arm64-{lite,full}.tar.gz` |
| `windows-x64`   | `windows`/`amd64` | `dist/{lite,full}/windows-x64/formattington.exe`   | `formattington-windows-x64-{lite,full}.zip`    |
| `windows-arm64` | `windows`/`arm64` | `dist/{lite,full}/windows-arm64/formattington.exe` | `formattington-windows-arm64-{lite,full}.zip`  |

Each archive contains a top-level `formattington-<platform>-{lite,full}/` directory holding the binary. The release version is stamped into every binary with `-ldflags -X`, so a binary downloaded from a release reports the version it was released as even outside the repository.

`FORMATTER_VERSION` controls which formatter release to fetch (default `latest`, resolved independently per tool). Forms that publish no suitable asset for a platform (for example `shfmt` and Windows arm64) are built anyway for the platforms they do support; missing tools are omitted from that platform's full binary and reported with a warning. Black's macOS x86_64 build (no upstream standalone asset) is compiled from source by `formatters/black/update.sh` on a native Intel Mac when one is available.

Before tagging, the freshly built full binary for the host is run under an empty `HOME` and a stripped `PATH`; every formatter embedded for the host platform's bundle is invoked with `--version`, which fails the release if the embedded formatters do not unpack and run on this host.

`make dist` runs the same fetch/build/package/smoke sequence without tagging or releasing.

## Continuous integration

There is no CI configuration in this repository yet. No workflow files exist under `.github/`, and nothing runs automatically on pushes or pull requests. The gate today is local: `make check` (`go vet ./...`, a `gofmt` verification, and `go test ./...`). Run it before pushing.

Self-hosted CI on macOS and Windows runners managed by [runner-spawner](https://github.com/fordaj/runner-spawner) (the bundled binary under `tools/runner-spawner/v<version>/<platform>/`, driven by the committed `.runner-spawner.json`) is planned and will be tracked as a separate ticket; when it lands, this section will describe the workflows, their labels, and their secrets handling. Until then, do not assume checks run on your PRs — if you contribute, run `make check` yourself and note the result in the pull request.

## Project status

Formattington is under active development. The core command-line interface and formatter integrations are being established, with additional languages, tools, distribution options, and workflow improvements planned.

## Tool sources and licenses

Formattington orchestrates third-party tools; they are never redistributed as part of the source distribution, and the bundled release binaries ship unmodified upstream executables with this attribution. Each tool belongs to its upstream project and remains under its own license:

| Tool | Source | License | Distribution |
| ---- | ------ | ------- | ------------ |
| prettier | https://github.com/prettier/prettier | MIT | Bundled |
| sqlfluff | https://github.com/sqlfluff/sqlfluff | MIT | PATH-only |
| black | https://github.com/psf/black | MIT | Bundled |
| MicroPython (via black) | https://github.com/psf/black | MIT | (bundled, as black) |
| shfmt | https://github.com/mvdan/sh | BSD-3-Clause | Bundled |
| google-java-format | https://github.com/google/google-java-format | Apache-2.0 | Bundled |
| dotnet-format | https://github.com/dotnet/sdk | MIT | PATH-only |
| powershell (PSScriptAnalyzer) | https://github.com/PowerShell/PSScriptAnalyzer | MIT | PATH-only |
| clang-format | https://github.com/cpp-linter/clang-tools-static-binaries (LLVM via https://github.com/llvm/llvm-project) | Unlicense (repackaged LLVM binaries; upstream LLVM is Apache-2.0 WITH LLVM-exception) | Bundled |
| rustfmt | https://github.com/rust-lang/rustfmt | Apache-2.0 | PATH-only |
| php-cs-fixer | https://github.com/PHP-CS-Fixer/PHP-CS-Fixer | MIT | PATH-only |
| gofmt | https://github.com/golang/go (cmd/gofmt) | BSD-3-Clause | PATH-only |
| goimports | https://github.com/golang/tools (cmd/goimports) | BSD-3-Clause | PATH-only |
| ktfmt | https://github.com/Kotlin/ktfmt | Apache-2.0 | Bundled |
| stylua | https://github.com/JohnnyMorganz/StyLua | MPL-2.0 | Bundled |
| swift-format | https://github.com/swiftlang/swift-format | Apache-2.0 | PATH-only |
| rubocop | https://github.com/rubocop/rubocop | MIT | PATH-only |
| dart-format | https://github.com/dart-lang/sdk | BSD-3-Clause | PATH-only |
| styler | https://github.com/r-lib/styler | MIT | PATH-only |
| nixfmt | https://github.com/NixOS/nixfmt | MPL-2.0 | PATH-only |
| gdformat | https://github.com/Scony/godot-gdscript-toolkit | MIT | PATH-only |
| npm-groovy-lint | https://github.com/nvuillam/npm-groovy-lint | MIT | PATH-only |
| perltidy | https://github.com/perltidy/perltidy | GPL-2.0+ | PATH-only, never embedded |
| zig-fmt | https://github.com/ziglang/zig | MIT | PATH-only |
| mix-format | https://github.com/elixir-lang/elixir | Apache-2.0 | PATH-only |
| fourmolu | https://github.com/fourmolu/fourmolu | BSD-3-Clause | Bundled |
| scalafmt | https://github.com/scalameta/scalafmt | Apache-2.0 | PATH-only |
| fprettify | https://github.com/pseewald/fprettify | GPL-3.0+ | PATH-only, never embedded |
| erlfmt | https://github.com/WhatsApp/erlfmt | Apache-2.0 | PATH-only |
| jlfmt | https://github.com/JuliaEditorSupport/JuliaFormatter.jl | MIT | PATH-only |
| cljfmt | https://github.com/weavejester/cljfmt | EPL-1.0 | Bundled (unmodified upstream binaries) |
| gleam-format | https://github.com/gleam-lang/gleam | Apache-2.0 | PATH-only |
| gnatformat | https://github.com/AdaCore/GNATformat | Apache-2.0 WITH LLVM-exception | PATH-only |
| fantomas | https://github.com/fsprojects/fantomas | Apache-2.0 | PATH-only |
| ocamlformat | https://github.com/ocaml-ppx/ocamlformat | MIT | PATH-only |
| forge-fmt | https://github.com/foundry-rs/foundry | Apache-2.0 | PATH-only |
| mojo-format | https://github.com/modular/mojo | Apache-2.0 WITH LLVM-exception (compiler portions); some MAX components under the Modular MAX License | PATH-only |
| odinfmt | https://github.com/DanielGavin/ols | MIT | PATH-only |
| elm-format | https://github.com/avh4/elm-format | BSD-3-Clause | Bundled |
| markdownlint | https://github.com/igorshubovych/markdownlint-cli | MIT | Bundled |
| taplo | https://github.com/tamasfe/taplo | MIT | Bundled |

Bundled tools are fetched from their upstream projects at release time and redistributed **unmodified**; the `formatters/<tool>/update.sh` scripts record the exact upstream location per tool. Bundled EPL-1.0 material (cljfmt) is redistributed unmodified, with this attribution. Tools available only from PATH are never redistributed by Formattington at all.

## License

See [LICENSE](LICENSE) for licensing information.

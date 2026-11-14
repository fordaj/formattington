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

Run formatters one after another (`shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `markdownlint`, `taplo`, `gofmt`, `goimports`, `stylua`, `nixfmt`, `php-cs-fixer`, `perltidy`, `fprettify`, `erlfmt`, `sqlfluff`, `gdformat`, `gdformat` order) instead of concurrently, for ordered output.

The flag is also accepted before `all`:

```bash
formattington --series all
formattington --check --series all
```

`--check` and `--dry-run` keep their usual meaning (they go before `all`): in check mode every formatter still reports, whether you use the concurrent default, `--series`, or both flags together.

### Choose a specific tool

```bash
formattington prettier
formattington clang-format
formattington shfmt
formattington black
formattington gofmt
formattington goimports
formattington stylua
formattington nixfmt
formattington php-cs-fixer
formattington perltidy
formattington fprettify
formattington erlfmt
formattington sqlfluff
formattington gdformat
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
formattington c
formattington cpp
formattington c++
formattington bash
formattington python
formattington md
formattington markdown
formattington html
formattington go
formattington lua
formattington nix
formattington php
formattington perl
formattington fortran
formattington erlang
formattington sql
formattington gdscript
```

Language selection lets you target the relevant files without having to remember which tool handles each extension: `c`, `cpp`, and `c++` resolve to `clang-format`; `bash` resolves to `shfmt`; `python`/`py` resolve to `black`; `md`/`markdown`/`html` resolve to `prettier`; `go` resolves to `gofmt`; `lua` resolves to `stylua`; `nix` resolves to `nixfmt`; `php` resolves to `php-cs-fixer`; and `perl` resolves to `perltidy`; and `fortran` resolves to `fprettify`; and `erlang` resolves to `erlfmt`; and `sql` resolves to `sqlfluff`; and `gdscript` resolves to `gdformat`. Use `goimports` when imports should also be fixed (for example `formattington goimports` or a config command listing `goimports`). Aliases accept the same `--check`/`--dry-run` flags (before the alias) and forwarded arguments (after it) as formatter names. Unrecognized words are a usage error (exit 2).

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

Each entry in `commands` is an argv list: element 0 is a formatter name (`shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `markdownlint`, `taplo`, `gofmt`, `goimports`, `stylua`, `nixfmt`, `php-cs-fixer`, `perltidy`, `fprettify`, `erlfmt`, `sqlfluff`, `gdformat`, `gdformat`) or a language alias (`bash`, `c`/`cpp`/`c++`, `python`/`py`, `md`/`markdown`/`html`, `ps1`, `toml`, `go`, `lua`, `nix`, `php`, `perl`, `fortran`, `erlang`, `sql`, `gdscript`); the remaining elements are forwarded verbatim to that formatter, exactly as arguments after the formatter name on the CLI (for example, `["black", "--check", "src"]` behaves like `formattington black --check src`). Arguments stay as discrete list entries, so paths with spaces are safe. When a command passes no explicit files, file discovery works as for single-tool runs. A sample is provided as `.formattington.example.json` (it is not auto-discovered).

Config discovery precedence (highest first):

| Priority | Source                                           | Notes                                                                                                                                              |
| -------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1        | `--config <path>` (before or after `all`)        | Explicit path, relative paths resolve against the current directory; `--config` after `all` (e.g. `formattington all --config x.json`) is honored. |
| 2        | `FORMATTINGTON_CONFIG_PATH` environment variable | Explicit path; an empty value counts as unset.                                                                                                     |
| 3        | `./.formattington.json`                          | Auto-discovered in the current directory only (no parent-directory or user-home search).                                                           |
| 4        | No config file                                   | `all` runs the full formatter registry as before (backwards compatible).                                                                           |

When a config is in effect, `all` runs exactly the listed commands. With no config, `all` keeps its current behavior (all eight formatters). The config only changes what `all` means: single-tool runs ignore the file and the environment variable, and `--config` before a single formatter (e.g. `formattington --config x.json black`) is a usage error (exit 2). `--config` after a single formatter is forwarded to that formatter, so native flags such as Prettier's `--config` keep working.

Global flags compose as with a full-registry `all`: `--check`/`--dry-run` and `--verbose` (or `FORMATTINGTON_VERBOSE=1`) propagate to every config command. Config commands run concurrently by default; use `--series` (before or after `all`) to run them sequentially in file order. The exit code is the last non-zero code in file order, or `0` only if every command succeeds. A missing formatter executable counts as `127` for that command and does not prevent the remaining commands from running.

Empty `commands: []` is valid; `all` then runs nothing and exits `0`.

Shell-string entries (`"black --check"`), object-form entries (`{"tool": "black", "args": [...]}`), and bare top-level arrays (`[["black"]]`) are rejected, as are unknown top-level keys (only `commands` is supported): the validator exits `2` with a `formattington: ...` message naming the file and the offending entry rather than misinterpreting them. Missing files, malformed JSON, and schema violations (missing/non-list `commands`, non-list/non-string/empty entries, unknown formatter/alias in element 0) all exit `2` the same way. A present but malformed `./.formattington.json` also exits `2` and does not silently fall back to the full registry.

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
| 0    | Success. Formatting applied (or nothing needed formatting).                                                                                                                                                                                   |
| 1    | Formatting changes required in check mode.                                                                                                                                                                                                    |
| 2    | Usage error (for example, unknown formatter, exclusive `--check` and `--dry-run` given together, unexpected arguments parsed by Formattington, or a missing/invalid `--config` / `FORMATTINGTON_CONFIG_PATH` / `./.formattington.json` file). |
| 127  | Formatter executable not found (no bundled binary and nothing on `PATH`).                                                                                                                                                                     |

### Mode semantics

Formattington owns the mode flag before the formatter name; arguments after the formatter are forwarded verbatim.

| Formatter    | `--check` (before formatter)                                                       | `--dry-run` (before formatter)                                                                 | Default                                                                                                               |
| ------------ | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| shfmt        | `shfmt -l` — list unformatted files; exit 1 if any listed, else 0                  | `shfmt -d` — print diff, do not modify                                                         | `shfmt -w` — write in place                                                                                           |
| clang-format | `clang-format --dry-run --Werror` — fail if any file needs formatting              | `clang-format --dry-run` — validate without modifying                                          | `clang-format -i` — format in place (unless `-i`/`--in-place` already given)                                          |
| black        | `black --check` — fail if any file needs formatting                                | `black --diff` — show diff, do not modify                                                      | `black` in place (no flag)                                                                                            |
| prettier     | `prettier --check` — fail if any file needs formatting                             | `prettier --list-different` — list files that differ, do not modify (Prettier has no `--diff`) | `prettier --write` — write in place (unless `--write`/`-w`, `--check`/`-c`, or `--list-different`/`-l` already given) |
| powershell   | PSScriptAnalyzer lint (no in-place formatting); findings exit 1                    | Same PSScriptAnalyzer lint invocation (check/dry-run are accepted for CLI uniformity)          | Same PSScriptAnalyzer lint invocation (findings exit 1)                                                               |
| taplo        | `taplo fmt --check` — fail if any file needs formatting                            | `taplo fmt --diff` — print diff, do not modify                                                 | `taplo fmt` in place (no flag)                                                                                        |
| gofmt        | `gofmt -l` — list unformatted files; exit 1 if any listed, else 0                  | `gofmt -d` — display diffs instead of rewriting files                                          | `gofmt -w` — write result to the source file instead of stdout (format in place)                                      |
| goimports    | `goimports -l` — list files whose formatting differs; exit 1 if any listed, else 0 | `goimports -d` — display diffs instead of rewriting files                                      | `goimports -w` — write result to the source file instead of stdout (format in place, fixing imports)                  |
| stylua       | `stylua --check` — fail if any file needs formatting; files are not modified                   | `stylua --check --output-format=unified` — print unified diffs, do not modify                    | `stylua` — format in place                                                                                          |
| nixfmt       | `nixfmt --check` — fail if any file needs formatting (exit 1; also used for parse errors)       | `nixfmt --check` (no diff mode exists) — validate without modifying                               | `nixfmt` — format in place                                                                                       |
| php-cs-fixer | `php-cs-fixer check` — fail if any file needs fixing (native exit 8 is reported as 1)              | `php-cs-fixer check` (non-modifying; same invocation)                                             | `php-cs-fixer fix` — fix in place                                                                               |
| perltidy     | `perltidy --assert-tidy` — exit 0 when tidy, non-zero (reported as 1) otherwise                  | `perltidy -d` — print a diff, do not modify                                                       | `perltidy -b -bext=/` — format in place (backups suppressed)                                                      |
| fprettify    | Synthesized: `fprettify --diff` — non-empty diff output means changes required (exit 1)          | `fprettify --diff` — print a diff, do not modify                                                  | `fprettify` — format in place                                                                                |
| erlfmt       | `erlfmt -c` — fail if any file needs formatting (exit 1)                                          | `erlfmt -c` (no diff mode exists) — validate without modifying                                      | `erlfmt -w` — format in place                                                                               |
| sqlfluff     | `sqlfluff lint --disable-progress-bar` — exit 1 on violations (exit 2 = error, passed through)       | `sqlfluff fix --dry-run --diff --disable-progress-bar` — print a diff, do not modify                  | `sqlfluff fix --disable-progress-bar` — fix in place                                                        |
| gdformat     | `gdformat -c` — print "would reformat <file>" and exit 1 when reformatting is needed              | `gdformat -d` — print a diff, do not modify (implies --check)                                           | `gdformat` — format in place                                                                                |

These mode mappings are integration-specific. Future tools may support different operations or native flags, and their integrations should preserve those differences.

### Missing executables

When a formatter executable cannot be found, Formattington does not crash with a stack trace. It prints a single line to stderr (for example:

```text
formattington: Could not find 'shfmt'. Expected a bundled executable under formatters/shfmt/<version>/<platform>/shfmt or an installation on PATH.
```

) and exits `127`.

This happens identically in default, `--check`, and `--dry-run` modes, for `shfmt`, `clang-format`, `black`, `prettier`, `powershell` (resolved as `pwsh`, then `powershell`, plus the PSScriptAnalyzer module), `taplo`, `gofmt`, `goimports`, `stylua`, `nixfmt`, `php-cs-fixer`, `perltidy`, `fprettify`, `erlfmt`, `sqlfluff`, `gdformat`, and `gdformat`, and under `all` (the remaining formatters still run in both the concurrent default and `--series`).

### Exit-code aggregation with `all`

The exit code is the **last non-zero code in `shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `markdownlint`, `taplo`, `gofmt`, `goimports`, `stylua`, `nixfmt`, `php-cs-fixer`, `perltidy`, `fprettify`, `erlfmt`, `sqlfluff`, `gdformat`, `gdformat` order** (a missing tool counts as `127` in its slot), or `0` only if every formatter succeeded.

The rule is deterministic regardless of completion order in the concurrent default.

For example:

- If `shfmt` returns `5` and `prettier` returns `1`, `all` exits `1`.
- If `shfmt` returns `5` and the rest succeed, `all` exits `5`.

With a config file, the same rule applies in config-command order: the last non-zero code among the listed commands wins (a missing tool counts as `127` in its command slot and does not stop later commands), or `0` only if every command succeeds. Invalid or missing config files exit `2` before any formatter runs.

Use `--verbose` to see which executable was resolved when a tool is present.

`formattington --help` and `formattington --version` do not require any formatter executables and continue to work when tools are absent.

### Install hints

- `shfmt` comes from [mvdan/sh](https://github.com/mvdan/sh).
- `clang-format` is part of LLVM ([documentation](https://clang.llvm.org/docs/ClangFormat.html)).
- `black` is installed via pip (`pip install black` or `uv tool install black`).
- `prettier` is installed via npm (`npm install -g prettier`).
- `powershell` lints with [PSScriptAnalyzer](https://github.com/microsoft/PSScriptAnalyzer) through PowerShell 7+ (`pwsh`); install the module with `Install-Module PSScriptAnalyzer` inside `pwsh` (Windows PowerShell 5.1, `powershell.exe`, is tried as a fallback).
- `taplo` (`taplo fmt`) formats TOML; a pinned copy is bundled in the full release archives under `formatters/taplo/<version>/<platform>/` for offline use (download the release binary and put it on `PATH` if you prefer a system install, or fetch one into your checkout with `formatters/taplo/update.sh`).
- `gofmt` is part of the [Go](https://go.dev/) installation (install Go for your OS; `gofmt` comes with it).
- `goimports` formats like `gofmt` and also fixes import grouping; install with `go install golang.org/x/tools/cmd/goimports@latest`.
- `stylua` is the standard Lua formatter; a pinned copy is bundled in the full release archives, or install with `brew install stylua` / `cargo install stylua`.
- `nixfmt` is the official Nix formatter (RFC 166); install via Nix (`nixpkgs.nixfmt-rfc-style`) or `brew install nixfmt`.
- `php-cs-fixer` is the de-facto PHP fixer; install with `composer global require friendsofphp/php-cs-fixer` or `brew install php-cs-fixer` (needs a PHP ≥ 7.4 interpreter on `PATH`). Its native check exit code is 8; Formattington reports it as 1.
- `perltidy` is the Perl formatter; install from CPAN (`cpan Perl::Tidy`) or your package manager. It is GPL-2.0+ and is never bundled into the release binaries (and requires `perl` on `PATH` — on Windows, install Strawberry Perl).
- `fprettify` is the de-facto Fortran formatter; install with `pipx install fprettify` / `pip install fprettify`. It is GPL-3.0+ and Python-based, so it was deliberately rejected for bundling; free-form Fortran only (`.f90`, `.f95`, `.f03`, `.f08`, `.f18`). Its `--diff` does not affect the exit code, so Formattington synthesizes check mode from non-empty diff output.
- `erlfmt` is the mature Erlang formatter; there are no bundleable binaries (it ships as a rebar3 plugin or escript), so it is PATH-only and needs a working Erlang/OTP 21+ install. Invoke the escript form (`erlfmt -w` / `erlfmt -c`).
- `sqlfluff` is the dialect-aware SQL linter/formatter; install with `pipx install sqlfluff` / `pip install sqlfluff`. There is no official binary and SQLFluff is Python-only, so it is PATH-only (a shared embedded Python runtime is tracked separately and was not introduced). Set the SQL dialect in the project config (`.sqlfluff` / `[tool.sqlfluff]` in `pyproject.toml`) or pass `--dialect` — Formattington never hardcodes one (SQLFluff falls back to `ansi` otherwise).
- `gdformat` is the Godot 4 GDScript formatter (from the gdtoolkit package); install with `pipx install gdtoolkit==4.*` (3.x targets the Godot 3 line). It is PATH-only; Formattington does not set `--fast`, so the safety checks stay on.

Either a bundled copy under `formatters/` (see [Bundled tools](#bundled-tools)) — or, in the full release builds, the copy embedded inside the binary — or an installation on `PATH` satisfies `FindTool()`. `gofmt` and `goimports` are PATH-only and are found via `PATH`.

## Supported tools

Formattington integrates existing tools rather than reimplementing their functionality.

| Tool                                                              | Category  | Status    | Typical use                                             |
| ----------------------------------------------------------------- | --------- | --------- | ------------------------------------------------------- |
| [clang-format](https://clang.llvm.org/docs/ClangFormat.html)      | Formatter | Available | C, C++, and related languages                           |
| [shfmt](https://github.com/mvdan/sh)                              | Formatter | Available | Shell scripts                                           |
| [Black](https://black.readthedocs.io/)                            | Formatter | Available | Python (`.py`)                                          |
| [Prettier](https://prettier.io/)                                  | Formatter | Available | JavaScript, TypeScript, JSON, HTML, CSS, Markdown, YAML |
| [PSScriptAnalyzer](https://github.com/microsoft/PSScriptAnalyzer) | Linter    | Available | PowerShell scripts (`.ps1`, `.psm1`) via `pwsh`         |
| [Taplo](https://taplo.tamasfe.dev/)                               | Formatter | Available | TOML (`.toml`) via `taplo fmt`                          |
| [gofmt](https://pkg.go.dev/cmd/gofmt)                             | Formatter | Available | Go (`.go`)                                              |
| [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)  | Formatter | Available | Go (`.go`) with import fixing                           |
| [markdownlint](https://github.com/DavidAnson/markdownlint)        | Linter    | Available | Markdown style and consistency via `markdownlint-cli`   |
| [StyLua](https://github.com/JohnnyMorganz/StyLua)                                 | Formatter | Available | Lua (`.lua`)                                            |
| [nixfmt](https://github.com/NixOS/nixfmt)                                         | Formatter | Available | Nix (`.nix`)                                            |
| [PHP-CS-Fixer](https://github.com/PHP-CS-Fixer/PHP-CS-Fixer)                       | Formatter | Available | PHP (`.php`) via `php-cs-fixer fix`                     |
| [perltidy](https://github.com/perltidy/perltidy)                              | Formatter | Available | Perl (`.pl`, `.pm`, `.t`) — PATH-only, GPL-2.0+            |
| [fprettify](https://github.com/pseewald/fprettify)                           | Formatter | Available | Fortran free-form (`.f90`…) — PATH-only, GPL-3.0+           |
| [erlfmt](https://github.com/WhatsApp/erlfmt)                                | Formatter | Available | Erlang (`.erl`, `.hrl`) — PATH-only; needs OTP 21+          |
| [SQLFluff](https://github.com/sqlfluff/sqlfluff)                          | Linter/Formatter | Available | SQL (`.sql`) via `sqlfluff fix` — PATH-only                 |
| [gdformat](https://github.com/Scony/godot-gdscript-toolkit)              | Formatter | Available | GDScript (`.gd`) — PATH-only, Godot 4                       |

Only `shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `markdownlint`, `taplo`, `gofmt`, `goimports`, `stylua`, `nixfmt`, `php-cs-fixer`, `perltidy`, `fprettify`, `erlfmt`, `sqlfluff`, `gdformat`, `c`/`cpp`/`c++` (aliases for `clang-format`), `bash` (alias for `shfmt`), `python`/`py` (aliases for `black`), `md`/`markdown`/`html` (aliases for `prettier`), `ps1` (alias for `powershell`), `toml` (alias for `taplo`), `go` (alias for `gofmt`), `lua` (alias for `stylua`), `nix` (alias for `nixfmt`), `php` (alias for `php-cs-fixer`), `perl` (alias for `perltidy`), `fortran` (alias for `fprettify`), `erlang` (alias for `erlfmt`), `sql` (alias for `sqlfluff`), `gdscript` (alias for `gdformat`), and `all` are accepted by the CLI.

Prettier is the sole owner of Markdown formatting. There is no dedicated Markdown formatter module; `formattington md` and `formattington markdown` resolve to the existing Prettier integration, which covers `.md` and `.markdown` files and respects Prettier's native configuration (such as `.prettierrc`). Markdown _style_ is a distinct operation and is linted by `formattington markdownlint`: it runs [`markdownlint-cli`](https://github.com/igorshubovych/markdownlint-cli) against `.md`/`.markdown` files, discovery batches files as usual, findings exit `1`, and no flags are injected — the tool is read-only without an explicit `--fix`, and `--fix`, `--config`, and glob patterns are forwarded verbatim, so it edits files only when you ask. Configuration is discovered natively by the tool (for example `.markdownlint.json` in the working directory) and `.prettierrc`'s sibling `.markdownlint.json` needs no Formattington-side setup. Aliases are not involved: use `md`/`markdown` for formatting and `markdownlint` for linting. Unrecognized names still report a usage error (exit 2).

Go has two formatters covering the same `.go` files: `gofmt` (canonical formatting) and `goimports` (formatting plus import grouping). `formattington go` resolves to `gofmt`; use `formattington goimports` or a config command listing `goimports` when imports should also be fixed. Use `--series` with `all` to run them sequentially (`gofmt` then `goimports`) instead of concurrently, or list only one in a config file to avoid running both against the same files. Per AGENTS.md, when multiple tools support the same file type, selection should respect configured tool choices.

Additional tools may provide other capabilities, such as checking, linting, and fixing, without necessarily being formatters.

### Prerequisites

| Tool         | Install method (bundled)                                                                                                                                                | Install method (PATH)                                                                                        | How Formattington finds it                                              |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| shfmt        | Fetched by `formatters/shfmt/update.sh`; bundled in the full release archives                                                                                           | `go install mvdan.cc/sh/v3/cmd/shfmt@latest` or your OS package manager                                      | Bundled (or embedded) first, then `PATH` via `FindTool("shfmt")`        |
| clang-format | Fetched by `formatters/clang-format/update.sh`; bundled in the full release archives                                                                                    | Install LLVM/clang-tools for your OS (for example `brew install clang-format` or `apt install clang-format`) | Bundled (or embedded) first, then `PATH` via `FindTool("clang-format")` |
| black        | Fetched by `formatters/black/update.sh` (its macOS x86_64 build is compiled from source by the same script on a native Intel Mac); bundled in the full release archives | `pip install black` or `uv tool install black`                                                               | Bundled (or embedded) first, then `PATH` via `FindTool("black")`        |
| prettier     | Fetched by `formatters/prettier/update.sh`; bundled in the full release archives                                                                                        | `npm install -g prettier`                                                                                    | Bundled (or embedded) first, then `PATH` via `FindTool("prettier")`     |
| powershell   | Not bundled (PATH-only; PSScriptAnalyzer is a module)                                                                                                                   | PowerShell 7+ (`pwsh`) on `PATH` plus `Install-Module PSScriptAnalyzer` in `pwsh`                            | `PATH` via `FindTool("pwsh")`, then `FindTool("powershell")`            |
| taplo        | Fetched by `formatters/taplo/update.sh`; bundled in the full release archives                                                                                           | Download taplo from https://github.com/tamasfe/taplo releases and put `taplo` on `PATH`                      | Bundled (or embedded) first, then `PATH` via `FindTool("taplo")`        |
| gofmt        | Not bundled (PATH-only)                                                                                                                                                 | Install Go for your OS; `gofmt` comes with the Go installation                                               | `PATH` via `FindTool("gofmt")`                                          |
| goimports    | Not bundled (PATH-only)                                                                                                                                                 | `go install golang.org/x/tools/cmd/goimports@latest` (ensure `$(go env GOPATH)/bin` is on `PATH`)            | `PATH` via `FindTool("goimports")`                                      |
| markdownlint | Fetched by `formatters/markdownlint/update.sh`; bundled in the full release archives                                                                        | `npm install -g markdownlint-cli`                                                                            | Bundled (or embedded) first, then `PATH` via `FindTool("markdownlint")` |
| stylua       | Fetched by `formatters/stylua/update.sh`; bundled in the full release archives (no Windows arm64 asset upstream)                                                          | `brew install stylua` or `cargo install stylua`                                                              | Bundled (or embedded) first, then `PATH` via `FindTool("stylua")`     |
| nixfmt       | Not bundled (PATH-only; upstream publishes only a static Linux x86_64 asset)                                                                                             | `nixpkgs.nixfmt-rfc-style` / `brew install nixfmt`                                                           | `PATH` via `FindTool("nixfmt")`                                      |
| php-cs-fixer | Not bundled (PATH-only; needs a PHP interpreter)                                                                                                                          | `composer global require friendsofphp/php-cs-fixer` / `brew install php-cs-fixer` (PHP ≥ 7.4 on `PATH`)        | `PATH` via `FindTool("php-cs-fixer")`                               |
| perltidy     | Not bundled (PATH-only; GPL-2.0+; needs `perl` on `PATH`)                                                                                                                | `cpan Perl::Tidy` / OS package manager (Windows: Strawberry Perl)                                              | `PATH` via `FindTool("perltidy")`                                  |
| fprettify    | Not bundled (PATH-only; GPL-3.0+; needs Python 3 on `PATH`)                                                                                                               | `pipx install fprettify` / `pip install fprettify`                                                             | `PATH` via `FindTool("fprettify")`                                  |
| erlfmt       | Not bundled (PATH-only; no bundleable binaries; needs Erlang/OTP 21+)                                                                                              | `brew install erlang` / OTP package of choice; escript built from rebar3 (`rebar3 as release escriptize`)   | `PATH` via `FindTool("erlfmt")`                                     |
| sqlfluff     | Not bundled (PATH-only; Python-only; no official binaries)                                                                                                             | `pipx install sqlfluff` / `pip install sqlfluff`                                                              | `PATH` via `FindTool("sqlfluff")`                                   |
| gdformat     | Not bundled (PATH-only; Python package; no official binaries)                                                                                                               | `pipx install gdtoolkit==4.*`                                                                      | `PATH` via `FindTool("gdformat")`                                    |

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

A fresh clone contains no formatter binaries (they are not committed); fetch them into your checkout with `formatters/<tool>/update.sh`, or download a `formattington-full-<platform>` release archive, which is a single self-contained binary with the formatters embedded.

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

The release `formattington-full-<platform>` binaries embed their platform's `formatters/` tree as a zip inside the executable (Go `full` build tag + `go:embed`). On first use the tree is unpacked into the per-user cache directory (`formatters/` under `os.UserCacheDir()`, for example `~/Library/Caches/formattington/formatters` on macOS) and reused from then on; upgrading to a binary with different embedded formatters replaces the extraction wholesale. The `formattington-lite-<platform>` binaries embed nothing.

Bundled tools are preferred; when none is found, lookup falls back to `PATH`.

`gofmt` and `goimports` are PATH-only and are not bundled; install `gofmt` via the Go installation, and `goimports` via `go install golang.org/x/tools/cmd/goimports@latest`, placing the executables on `PATH`.

`markdownlint` is bundled in the full release archives, wired into the CLI (registration order: after `powershell`), and included in `all` like every other registered tool. `stylua` is bundled the same way (registration order: after `goimports`; no Windows arm64 asset is published upstream, so that platform relies on `PATH`).

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

Two archives are produced per platform — `formattington-full-<platform>` and `formattington-lite-<platform>`. The **full** binary embeds the formatter executables for its target platform inside itself (Go `full` build tag + `go:embed`, as a zip unpacked into the per-user cache on first use), so it is a single self-contained executable; the **lite** binary is just `formattington`. All formatter integrations resolve their bundled copies through `FormattersDir()` at runtime: the `FORMATTINGTON_FORMATTERS_DIR` environment variable wins, then the embedded formatters (full builds), then a `formatters/` directory next to the executable or in an ancestor directory (a repository checkout), then `PATH`.

The cross-compile matrix (CGO disabled, so the binaries are statically linked and portable):

| Platform        | `GOOS`/`GOARCH`   | Binary in `dist/`                                  | Release asset                                  |
| --------------- | ----------------- | -------------------------------------------------- | ---------------------------------------------- |
| `linux-x64`     | `linux`/`amd64`   | `dist/{lite,full}/linux-x64/formattington`         | `formattington-{lite,full}-linux-x64.tar.gz`   |
| `linux-arm64`   | `linux`/`arm64`   | `dist/{lite,full}/linux-arm64/formattington`       | `formattington-{lite,full}-linux-arm64.tar.gz` |
| `macos-x64`     | `darwin`/`amd64`  | `dist/{lite,full}/macos-x64/formattington`         | `formattington-{lite,full}-macos-x64.tar.gz`   |
| `macos-arm64`   | `darwin`/`arm64`  | `dist/{lite,full}/macos-arm64/formattington`       | `formattington-{lite,full}-macos-arm64.tar.gz` |
| `windows-x64`   | `windows`/`amd64` | `dist/{lite,full}/windows-x64/formattington.exe`   | `formattington-{lite,full}-windows-x64.zip`    |
| `windows-arm64` | `windows`/`arm64` | `dist/{lite,full}/windows-arm64/formattington.exe` | `formattington-{lite,full}-windows-arm64.zip`  |

Each archive contains a top-level `formattington-{lite,full}-<platform>/` directory holding the binary. The release version is stamped into every binary with `-ldflags -X`, so a binary downloaded from a release reports the version it was released as even outside the repository.

`FORMATTER_VERSION` controls which formatter release to fetch (default `latest`, resolved independently per tool). Forms that publish no suitable asset for a platform (for example `shfmt` and Windows arm64) are built anyway for the platforms they do support; missing tools are omitted from that platform's full binary and reported with a warning. Black's macOS x86_64 build (no upstream standalone asset) is compiled from source by `formatters/black/update.sh` on a native Intel Mac when one is available.

Before tagging, the freshly built full binary for the host is run under an empty `HOME` and a stripped `PATH`; every formatter embedded for the host platform's bundle is invoked with `--version`, which fails the release if the embedded formatters do not unpack and run on this host.

`make dist` runs the same fetch/build/package/smoke sequence without tagging or releasing.

## Continuous integration

CI runs on self-hosted runners managed by [runner-spawner](https://github.com/fordaj/runner-spawner): the bundled binary under `tools/runner-spawner/v<version>/<platform>/` installs repository-scoped runners at `$HOME/.runners/formattington/` (one checkout per run) with auto labels `self-hosted`, `<os>` (`windows`/`macos`), arch, hostname, and `<hostname>-<runner-name>`. The workflows under `.github/` (callers `pull_request_caller.yml` and `nightly_caller.yml`, reusable `pull_request.yml`/`nightly.yml`) use those labels to pin `Setup` (checkout) and `Checks` (format + tests) to the same host.

Host setup:

```bash
source scripts/entry.sh
"$RUNNER_SPAWNER_BIN" list                  # configured names + on-disk dirs
"$RUNNER_SPAWNER_BIN" create --dry-run      # preview labels and paths
"$RUNNER_SPAWNER_BIN" create                 # needs a registration token (RUNNER_SPAWNER_TOKEN / GH_TOKEN / gh)
```

The committed `.runner-spawner.json` (owner `fordaj`, repo `formattington`, `runnerNames: ["runner01"]`) drives the default; edit it per host for more runners. A caller-provided `RUNNER_SPAWNER_BIN` wins over the bundle. `create` downloads the actions runner archive once into `.runners/cache` and installs the service; on Windows it must run with admin privileges and `--windows-username`/`--windows-password` (or the env vars); on macOS it installs via `svc.sh`.

Go 1.24+ must be on `PATH` on each host — the Checks step runs `make check` (which builds the binary, vets it, and runs `go test ./...`; the suite is the primary gate). No formatter binaries are committed to the repository; `formattington --check all` exits `127` on a fresh checkout unless the tools are installed on the host or fetched with `formatters/<tool>/update.sh` (or the check is run through a `formattington-full-<platform>` release binary, which carries the formatters inside itself). `pull_request_caller.yml` and `nightly_caller.yml` pass secrets with `secrets: inherit` so the reusable workflow's checkout can clone with `GITHUB_TOKEN`. Linux is not yet in the OS matrices — runner-spawner supports macOS and Windows runners only (Linux binaries ship for `list`/`status` diagnostics).

## Project status

Formattington is under active development. The core command-line interface and formatter integrations are being established, with additional languages, tools, distribution options, and workflow improvements planned.

## License

See [LICENSE](LICENSE) for licensing information.

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

Run formatters one after another (`shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `taplo`, `gofmt`, `goimports` order) instead of concurrently, for ordered output.

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

Discovery skips hidden directories and the usual generated/vendored locations (`__pycache__`, `node_modules`, `vendor`, `venv`, `dist`, `build`, and the bundled-tools directory `tools/`). If you pass explicit file arguments (for example, `formattington shfmt myfile.sh`), discovery is skipped and only those files are processed. Informational passthroughs (`--help`, `--version`) never trigger discovery.

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
```

Language selection lets you target the relevant files without having to remember which tool handles each extension: `c`, `cpp`, and `c++` resolve to `clang-format`; `bash` resolves to `shfmt`; `python`/`py` resolve to `black`; `md`/`markdown`/`html` resolve to `prettier`; and `go` resolves to `gofmt`. Use `goimports` when imports should also be fixed (for example `formattington goimports` or a config command listing `goimports`). Aliases accept the same `--check`/`--dry-run` flags (before the alias) and forwarded arguments (after it) as formatter names. Unrecognized words are a usage error (exit 2).

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

Each entry in `commands` is an argv list: element 0 is a formatter name (`shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `taplo`, `gofmt`, `goimports`) or a language alias (`bash`, `c`/`cpp`/`c++`, `python`/`py`, `md`/`markdown`/`html`, `ps1`, `toml`, `go`); the remaining elements are forwarded verbatim to that formatter, exactly as arguments after the formatter name on the CLI (for example, `["black", "--check", "src"]` behaves like `formattington black --check src`). Arguments stay as discrete list entries, so paths with spaces are safe. When a command passes no explicit files, file discovery works as for single-tool runs. A sample is provided as `.formattington.example.json` (it is not auto-discovered).

Config discovery precedence (highest first):

| Priority | Source | Notes |
| -------- | ------ | ----- |
| 1 | `--config <path>` (before or after `all`) | Explicit path, relative paths resolve against the current directory; `--config` after `all` (e.g. `formattington all --config x.json`) is honored. |
| 2 | `FORMATTINGTON_CONFIG_PATH` environment variable | Explicit path; an empty value counts as unset. |
| 3 | `./.formattington.json` | Auto-discovered in the current directory only (no parent-directory or user-home search). |
| 4 | No config file | `all` runs the full formatter registry as before (backwards compatible). |

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

| Code | Meaning                                                                                                                                            |
| ---- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0    | Success. Formatting applied (or nothing needed formatting).                                                                                        |
| 1    | Formatting changes required in check mode.                                                                                                         |
| 2    | Usage error (for example, unknown formatter, exclusive `--check` and `--dry-run` given together, unexpected arguments parsed by Formattington, or a missing/invalid `--config` / `FORMATTINGTON_CONFIG_PATH` / `./.formattington.json` file). |
| 127  | Formatter executable not found (no bundled binary and nothing on `PATH`).                                                                          |

### Mode semantics

Formattington owns the mode flag before the formatter name; arguments after the formatter are forwarded verbatim.

| Formatter    | `--check` (before formatter)                                          | `--dry-run` (before formatter)                                                                 | Default                                                                                                               |
| ------------ | --------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| shfmt        | `shfmt -l` — list unformatted files; exit 1 if any listed, else 0     | `shfmt -d` — print diff, do not modify                                                         | `shfmt -w` — write in place                                                                                           |
| clang-format | `clang-format --dry-run --Werror` — fail if any file needs formatting | `clang-format --dry-run` — validate without modifying                                          | `clang-format -i` — format in place (unless `-i`/`--in-place` already given)                                          |
| black        | `black --check` — fail if any file needs formatting                   | `black --diff` — show diff, do not modify                                                      | `black` in place (no flag)                                                                                            |
| prettier     | `prettier --check` — fail if any file needs formatting                | `prettier --list-different` — list files that differ, do not modify (Prettier has no `--diff`) | `prettier --write` — write in place (unless `--write`/`-w`, `--check`/`-c`, or `--list-different`/`-l` already given) |
| powershell   | PSScriptAnalyzer lint (no in-place formatting); findings exit 1       | Same PSScriptAnalyzer lint invocation (check/dry-run are accepted for CLI uniformity)            | Same PSScriptAnalyzer lint invocation (findings exit 1)                                                                                    |
| taplo        | `taplo fmt --check` — fail if any file needs formatting               | `taplo fmt --diff` — print diff, do not modify                         | `taplo fmt` in place (no flag)                                                                                   |
| gofmt        | `gofmt -l` — list unformatted files; exit 1 if any listed, else 0     | `gofmt -d` — display diffs instead of rewriting files                                          | `gofmt -w` — write result to the source file instead of stdout (format in place)                                    |
| goimports    | `goimports -l` — list files whose formatting differs; exit 1 if any listed, else 0 | `goimports -d` — display diffs instead of rewriting files                          | `goimports -w` — write result to the source file instead of stdout (format in place, fixing imports)                |

These mode mappings are integration-specific. Future tools may support different operations or native flags, and their integrations should preserve those differences.

### Missing executables

When a formatter executable cannot be found, Formattington does not crash with a stack trace. It prints a single line to stderr (for example:

```text
formattington: Could not find 'shfmt'. Expected a bundled executable under tools/shfmt/<version>/<platform>/shfmt or an installation on PATH.
```

) and exits `127`.

This happens identically in default, `--check`, and `--dry-run` modes, for `shfmt`, `clang-format`, `black`, `prettier`, `powershell` (resolved as `pwsh`, then `powershell`, plus the PSScriptAnalyzer module), `taplo`, `gofmt`, and `goimports`, and under `all` (the remaining formatters still run in both the concurrent default and `--series`).

### Exit-code aggregation with `all`

The exit code is the **last non-zero code in `shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `taplo`, `gofmt`, `goimports` order** (a missing tool counts as `127` in its slot), or `0` only if every formatter succeeded.

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
- `taplo` (`taplo fmt`) formats TOML; a pinned copy is bundled under `tools/taplo/v0.10.0/<platform>/` for offline use (download the release binary and put it on `PATH` if you prefer a system install).
- `gofmt` is part of the [Go](https://go.dev/) installation (install Go for your OS; `gofmt` comes with it).
- `goimports` formats like `gofmt` and also fixes import grouping; install with `go install golang.org/x/tools/cmd/goimports@latest`.

Either a bundled copy under `tools/` (see [Bundled tools](#bundled-tools)) or an installation on `PATH` satisfies `FindTool()`. `black`, `prettier`, `gofmt`, and `goimports` are PATH-only (not bundled) and are found via `PATH`.

## Supported tools

Formattington integrates existing tools rather than reimplementing their functionality.

| Tool                                                         | Category  | Status    | Typical use                                             |
| ------------------------------------------------------------ | --------- | --------- | ------------------------------------------------------- |
| [clang-format](https://clang.llvm.org/docs/ClangFormat.html) | Formatter | Available | C, C++, and related languages                           |
| [shfmt](https://github.com/mvdan/sh)                         | Formatter | Available | Shell scripts                                           |
| [Black](https://black.readthedocs.io/)                       | Formatter | Available | Python (`.py`)                                          |
| [Prettier](https://prettier.io/)                             | Formatter | Available | JavaScript, TypeScript, JSON, HTML, CSS, Markdown, YAML |
| [PSScriptAnalyzer](https://github.com/microsoft/PSScriptAnalyzer) | Linter | Available | PowerShell scripts (`.ps1`, `.psm1`) via `pwsh` |
| [Taplo](https://taplo.tamasfe.dev/)                            | Formatter | Available | TOML (`.toml`) via `taplo fmt`                          |
| [gofmt](https://pkg.go.dev/cmd/gofmt)                        | Formatter | Available | Go (`.go`)                                              |
| [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) | Formatter | Available | Go (`.go`) with import fixing                           |
| [markdownlint](https://github.com/DavidAnson/markdownlint)   | Linter    | Planned   | Markdown style and consistency                          |

Only `shfmt`, `clang-format`, `black`, `prettier`, `powershell`, `taplo`, `gofmt`, `goimports`, `c`/`cpp`/`c++` (aliases for `clang-format`), `bash` (alias for `shfmt`), `python`/`py` (aliases for `black`), `md`/`markdown`/`html` (aliases for `prettier`), `ps1` (alias for `powershell`), `toml` (alias for `taplo`), `go` (alias for `gofmt`), and `all` are accepted by the CLI.

Prettier is the sole owner of Markdown formatting. There is no dedicated Markdown formatter module; `formattington md` and `formattington markdown` resolve to the existing Prettier integration, which covers `.md` and `.markdown` files and respects Prettier's native configuration (such as `.prettierrc`). Markdown-specific linting (for example `markdownlint`) is a distinct operation and is out of scope for this change; it can be integrated separately in the future if needed. Unrecognized names still report a usage error (exit 2).

Go has two formatters covering the same `.go` files: `gofmt` (canonical formatting) and `goimports` (formatting plus import grouping). `formattington go` resolves to `gofmt`; use `formattington goimports` or a config command listing `goimports` when imports should also be fixed. Use `--series` with `all` to run them sequentially (`gofmt` then `goimports`) instead of concurrently, or list only one in a config file to avoid running both against the same files. Per AGENTS.md, when multiple tools support the same file type, selection should respect configured tool choices.

Additional tools may provide other capabilities, such as checking, linting, and fixing, without necessarily being formatters.

### Prerequisites

| Tool         | Install method (bundled)                                                          | Install method (PATH)                                                                                        | How Formattington finds it                                 |
| ------------ | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------- |
| shfmt        | Comes under `tools/shfmt/v3.14.1/<platform>/shfmt` in a fresh clone               | `go install mvdan.cc/sh/v3/cmd/shfmt@latest` or your OS package manager                                      | Bundled first, then `PATH` via `FindTool("shfmt")`        |
| clang-format | Comes under `tools/clang-format/v23.1.3/<platform>/clang-format` in a fresh clone | Install LLVM/clang-tools for your OS (for example `brew install clang-format` or `apt install clang-format`) | Bundled first, then `PATH` via `FindTool("clang-format")` |
| black        | Not bundled (PATH-only)                                                           | `pip install black` or `uv tool install black`                                                               | `PATH` via `FindTool("black")`                            |
| prettier     | Not bundled (PATH-only)                                                           | `npm install -g prettier`                                                                                    | `PATH` via `FindTool("prettier")`                         |
| powershell   | Not bundled (PATH-only; PSScriptAnalyzer is a module)                             | PowerShell 7+ (`pwsh`) on `PATH` plus `Install-Module PSScriptAnalyzer` in `pwsh`                            | `PATH` via `FindTool("pwsh")`, then `FindTool("powershell")` |
| taplo        | Comes under `tools/taplo/v0.10.0/<platform>/taplo` in a fresh clone               | Download taplo from https://github.com/tamasfe/taplo releases and put `taplo` on `PATH`                     | Bundled first, then `PATH` via `FindTool("taplo")`        |
| gofmt        | Not bundled (PATH-only)                                                           | Install Go for your OS; `gofmt` comes with the Go installation                                               | `PATH` via `FindTool("gofmt")`                            |
| goimports    | Not bundled (PATH-only)                                                           | `go install golang.org/x/tools/cmd/goimports@latest` (ensure `$(go env GOPATH)/bin` is on `PATH`)           | `PATH` via `FindTool("goimports")`                        |

Bundled versions are pinned in `tools/` for offline use; see [Bundled tools](#bundled-tools) for the layout and refresh process.

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

A fresh clone already contains the pinned bundled executables under `tools/`, so building the binary is sufficient for offline use.

If you prefer a system installation, put the tool on `PATH` instead — `FindTool()` prefers bundled copies and falls back to `PATH`.

`black` is PATH-only (installed via `pip install black` or `uv tool install black`) and is not bundled under `tools/`.

`prettier` is PATH-only (installed via `npm install -g prettier`) and is not bundled under `tools/`.

`gofmt` (part of the Go installation) and `goimports` (installed via `go install golang.org/x/tools/cmd/goimports@latest`) are PATH-only and are not bundled under `tools/`.

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
tools/<name>/v<version>/<platform>/
```

Here, `<platform>` is the hyphenated `<os>-<arch>` name returned by `PlatformDir()` (for example `linux-x86_64`, `macos-arm64`, `windows-x86_64`).

Bundled tools are preferred; when none is found, lookup falls back to `PATH`.

`black`, `prettier`, `gofmt`, and `goimports` are PATH-only and are not bundled under `tools/`; install `black` via `pip install black` or `uv tool install black`, `prettier` via `npm install -g prettier`, `gofmt` via the Go installation, and `goimports` via `go install golang.org/x/tools/cmd/goimports@latest`, placing the executables on `PATH`.

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

### Binary policy

Formattington intentionally commits pinned formatter binaries under `tools/` (option (a) in #11).

Bundled binaries are the first lookup location in `FindTool()` and the fallback is `PATH`, so deleting or updating a bundle never breaks the fallback — a missing bundle simply defers to `PATH` and, if that also fails, exits `127` with a one-line `formattington: Could not find ...` message.

Pinned upstream versions (recorded in `tools/README.md`):

- `shfmt` `v3.14.1` from [mvdan/sh](https://github.com/mvdan/sh).
- `clang-format` `v23.1.3` (plus a retained legacy `v22.1.8` macOS bundle) from the LLVM project.
- `taplo` `v0.10.0` from [tamasfe/taplo](https://github.com/tamasfe/taplo) (taplo-cli).
- `runner-spawner` `v0.0.1.0.4` from [fordaj/runner-spawner](https://github.com/fordaj/runner-spawner) (not a formatter; bundled for CI runner management, see "Continuous integration").

Keep these exact versions in sync with the docs when refreshing.

To refresh, download the new upstream release for each platform, extract with `tools/clang-format/unzip.sh` where applicable (taplo ships per-platform gzipped binaries; gunzip to `taplo`/`taplo.exe`), preserve the `tools/<name>/v<version>/<platform>/` layout, and verify with `formattington --verbose shfmt` (or `FORMATTINGTON_VERBOSE=1`).

There is no automatic fetch on first run; this keeps offline use reliable at the cost of a large clone (about 120 MB for `tools/`).

See `tools/README.md` for the full layout and version table.

### Repository hygiene

`bin/`, `*.exe`, `*.test`, `*.out`, `.DS_Store`, and `.logs/` are ignored via `.gitignore` and must not be tracked.

The previously committed `__pycache__/*.pyc` and `tools/**/*.DS_Store` files were untracked with `git rm --cached`; only the formatter binaries remain tracked under `tools/`.

## Testing

Run the test suite with:

```bash
make test
```

That is a thin wrapper around `go test ./...`. Tests live next to the code in `internal/formattington/` and cover CLI parsing, tool discovery, subprocess boundaries, argument forwarding, and the configuration file. They run without network access and without formatter binaries installed, using fake executable scripts in temporary directories.

`make check` additionally runs `go vet ./...` and verifies that `gofmt` has nothing to change.

## Releases

`release.bash` computes the next version with [versioner](https://github.com/fordaj/Versioner) (the bundled binary, selected via `scripts/entry.sh`), cross-compiles the program for every supported platform, tags the commit, pushes the tag, and creates a GitHub release with the per-platform archives attached.

```bash
./release.bash                # version, build, tag, push, release + upload
./release.bash --build-only   # cross-compile into dist/ only
./release.bash --dry-run      # compute + verify + build, touch nothing remote
```

The cross-compile matrix (CGO disabled, so the binaries are statically linked and portable):

| Platform      | `GOOS`/`GOARCH` | Binary in `dist/`              | Release asset                        |
| ------------- | --------------- | ------------------------------ | ------------------------------------ |
| `linux-x64`   | `linux`/`amd64` | `dist/linux-x64/formattington` | `formattington-linux-x64.tar.gz`     |
| `linux-arm64` | `linux`/`arm64` | `dist/linux-arm64/formattington` | `formattington-linux-arm64.tar.gz` |
| `macos-x64`   | `darwin`/`amd64` | `dist/macos-x64/formattington` | `formattington-macos-x64.tar.gz`     |
| `macos-arm64` | `darwin`/`arm64` | `dist/macos-arm64/formattington` | `formattington-macos-arm64.tar.gz` |
| `windows-x64` | `windows`/`amd64` | `dist/windows-x64/formattington.exe` | `formattington-windows-x64.zip` |
| `windows-arm64` | `windows`/`arm64` | `dist/windows-arm64/formattington.exe` | `formattington-windows-arm64.zip` |

Each archive contains the platform directory with its binary. The release version is stamped into every binary with `-ldflags -X`, so a binary downloaded from a release reports the version it was released as even outside the repository.

`make dist` runs the same cross-compilation without tagging or releasing.

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

Go 1.24+ must be on `PATH` on each host — the Checks step runs `make check` (which builds the binary, vets it, and runs `go test ./...`; the suite is the primary gate). Because `black`, `prettier`, `gofmt`, and `goimports` are PATH-only (not bundled), `formattington --check all` exits `127` unless those tools are installed on the host; the bundled `shfmt`, `clang-format`, and `taplo` work offline from `tools/`. `pull_request_caller.yml` and `nightly_caller.yml` pass secrets with `secrets: inherit` so the reusable workflow's checkout can clone with `GITHUB_TOKEN`. Linux is not yet in the OS matrices — runner-spawner supports macOS and Windows runners only (Linux binaries ship for `list`/`status` diagnostics).

## Project status

Formattington is under active development. The core command-line interface and formatter integrations are being established, with additional languages, tools, distribution options, and workflow improvements planned.

## License

See [LICENSE](LICENSE) for licensing information.
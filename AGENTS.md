# AGENTS.md

## Project Overview

Formattington is a cross-platform, multi-language formatting and code-quality tool orchestrator. It provides a unified command-line interface for invoking existing tools across multidisciplinary projects.

The goal is to bring project formatting and automated quality checks under one interface without reimplementing the underlying tools.

Formattington is written in Go and has no third-party dependencies.

## Core Principles

- **Reuse existing tools.** Delegate formatting, linting, and analysis to established tools rather than implementing language-specific logic.
- **Provide a consistent interface.** Users should not need to remember every tool's command-line syntax.
- **Support multidisciplinary projects.** A single repository may contain many languages and file types.
- **Keep orchestration lightweight.** Avoid unnecessary work, subprocess launches, and dependencies.
- **Preserve native configuration.** Respect tool-specific configuration files and repository conventions.
- **Remain cross-platform.** Account for Windows, macOS, and Linux differences in paths, executable discovery, and subprocess behavior.
- **Make failures visible.** Preserve meaningful exit statuses and diagnostics from underlying tools.
- **Respect tool capabilities.** Do not assume every integration supports the same operations or behavior.

## Core Terminology

Formattington orchestrates external tools that may provide different operations.

- **Tool:** An external executable or application integrated into Formattington.
- **Integration:** The code responsible for invoking and managing a particular tool.
- **Operation:** An action supported by a tool, such as formatting, checking, linting, or fixing.
- **File mapping:** The relationship between file types and the tools that can process them.

Common operations include:

| Operation | Purpose                                                          |
| --------- | ---------------------------------------------------------------- |
| Format    | Rewrite files into a consistent representation.                  |
| Check     | Identify formatting or style violations without modifying files. |
| Lint      | Analyze files for rule violations and report findings.           |
| Fix       | Automatically resolve issues supported by the underlying tool.   |

Not every tool supports every operation.

For example, `markdownlint --fix` can automatically correct certain Markdown violations but cannot resolve every linting issue.

Formatting and linting are complementary. A file can be correctly formatted while still containing lint violations.

Do not force tools into a common feature set when their native capabilities differ.

## Command-Line Interface

The established public interface supports three selection modes, plus the informational `list` and `init` commands.

### All configured tools

```bash
formattington all
formattington all --series
```

- `all` runs all configured formatter integrations against their matching files.
- By default, formatters run concurrently via goroutines.
- `--series` (before or after `all`, e.g. `formattington all --series`) requests sequential execution in registration order.
- Exit code is the last non-zero code in `prettier`, `sqlfluff`, `black`, `shfmt`, `google-java-format`, `dotnet-format`, `powershell`, `clang-format`, `rustfmt`, `php-cs-fixer`, `gofmt`, `goimports`, `ktfmt`, `stylua`, `swift-format`, `rubocop`, `dart-format`, `styler`, `nixfmt`, `gdformat`, `npm-groovy-lint`, `perltidy`, `zig-fmt`, `mix-format`, `fourmolu`, `scalafmt`, `fprettify`, `erlfmt`, `jlfmt`, `cljfmt`, `gleam-format`, `gnatformat`, `fantomas`, `ocamlformat`, `forge-fmt`, `mojo-format`, `odinfmt`, `elm-format`, `markdownlint`, `taplo` order (SO Survey 2026 popularity order; missing tool = 127); `0` only if all succeed.
- A missing or failing formatter never prevents the others from running.

Until explicitly changed, `all` remains formatter-focused. Do not automatically introduce linting, fixing, or other operations into its execution.

### Specific tool

```bash
formattington prettier
formattington shfmt
formattington clang-format
```

Arguments following a tool selection should be forwarded to that tool where supported.

Examples:

```bash
formattington prettier --check
formattington shfmt -d
formattington clang-format --dry-run
```

Formattington should preserve native tool arguments and behavior rather than unnecessarily recreating their options.

Future integrations may expose additional tools and operations without requiring every integration to support the same command-line arguments.

### Language or file type

```bash
formattington c
formattington cpp
formattington c++
formattington bash
formattington md
formattington markdown
formattington html
```

Language aliases should resolve to the appropriate tool integration and matching file types.

All documented names above resolve: `md`/`markdown`/`html` → `prettier`, `python`/`py` → `black`, `bash` → `shfmt`, `c`/`cpp`/`c++` → `clang-format`. Prettier is the sole owner of Markdown formatting: `md`/`markdown` reuse the `prettier` dispatch and its `.prettierrc` handling, with no separate Markdown module. Unrecognized words produce a usage error (exit 2); an alias whose backend is not registered yet reports `'<name>' is not yet supported` (exit 2) rather than silently succeeding.

When multiple tools support the same file type, selection should respect configured tool choices rather than arbitrarily invoking conflicting tools.

When modifying CLI behavior, preserve these public command patterns unless the change explicitly updates the interface.

### List registered tools

```bash
formattington list
```

- `list` prints a pipe-separated table of every registered tool with its Bundled and PATH versions (`-` when either side does not exist).
- The bundled side reports the newest host-platform bundle `FindTool` would resolve (or the one unpacked from a full release binary's embedded payload); the PATH side resolves the same executable the integration resolves and probes it at list time (`--version` where supported; `gofmt`/`goimports` via `go version <path>`, `zig` via `zig version`, `dotnet` via `dotnet format --version`, `Rscript` for the installed styler package version).
- Probes are bounded by a short timeout and never modify files; a missing executable or an undeterminable version reports `-`.
- `list` takes no arguments or mode flags — `--check`/`--dry-run`/`--series`/`--config` are usage errors (exit 2) — and exits 0.

### Initialize a project

```bash
formattington init
```

- `init` examines the current directory recursively (`git ls-files` — preferred, including submodules — plus untracked files that are not excluded by `.gitignore`, inside a Git repository; every regular file otherwise; hidden directories and the usual generated/vendored/package-manager directory names are always skipped) and writes `./.formattington.json` with the formatter invocations for the file types present, drawn from the embedded catalogue `internal/formattington/init.json` (one command per detected registry tool, in registration order, no file arguments — discovery behaves as usual).
- A repository encountered during the scan is examined from its own tracked perspective; tracked files deleted from the working tree are not counted.
- If `./.formattington.json` already exists, `init` asks for confirmation first (`y`/`yes` overwrites); a non-interactive shell refuses without modifying the file.
- `init` takes no arguments or mode flags — `--check`/`--dry-run`/`--series`/`--config` are usage errors (exit 2) — prints `Examining current directory... found N files across M file extensions.`, and exits 0 on success (2 on usage errors or refusal to overwrite).

## Architecture

The program lives in `cmd/formattington/` (entry point) and `internal/formattington/` (the whole implementation).

Important files include:

- `cmd/formattington/main.go` — process entry point; calls `formattington.Main(os.Args[1:])` and exits with its return code.
- `internal/formattington/cli.go` — canonical command-line entry point and orchestration (sole `registry` of tools; `languageAliases` map).
- `internal/formattington/list.go` — the `list` command; reports each registered tool's bundled and PATH versions.
- `internal/formattington/init.go` — the `init` command; examines the current directory (Git-aware) and writes `./.formattington.json` from the embedded catalogue.
- `internal/formattington/init.json` — the embedded catalogue `init` draws from: one command per registered tool, in registration order. Mirrors the registry exactly (like the hardcoded order lists in the docs). Validated and round-tripped by `internal/formattington/template_test.go`.
- `internal/formattington/tools.go` — shared tool discovery (`FindTool`, `FormattersDir`), platform detection (`PlatformDir`), and subprocess helpers (`runTool`).
- `internal/formattington/embedded.go` — unpacks the formatter executables embedded in the "full" release binary into the per-user cache directory on first use.
- `internal/formattington/embedded_full.go` / `internal/formattington/embedded_lite.go` — the `go:embed` payload (build tag `full`) and its lite-build stub.
- `internal/formattington/discovery.go` — recursive file discovery (`DiscoverFiles`).
- `internal/formattington/config.go` — `.formattington.json` loading and validation.
- `internal/formattington/help.go` — usage text and the help output.
- `internal/formattington/version.go` — dynamic version resolution (versioner, `FORMATTINGTON_VERSION`).
- `internal/formattington/formatter.go` — the `Formatter` and `Options` types shared by the integrations.
- `internal/formattington/prettier.go` — Prettier integration (`.js`, `.ts`, `.tsx`, `.jsx`, `.json`, `.html`, `.css`, `.scss`, `.yaml`, `.yml`, `.md`, `.markdown`; resolved from PATH or the bundled formatters; respects `.prettierrc`). Prettier is the sole owner of Markdown formatting; `md`/`markdown` are entries in `languageAliases` for `prettier`, and there is no separate Markdown module.
- `internal/formattington/sqlfluff.go` — SQLFluff (SQL) integration (`.sql` files; PATH-only). The `sql` language alias resolves to `sqlfluff` in `cli.go`. `sqlfluff lint` exit 1 is reported as `ExitFormatRequired`; exit 2 (error) passes through. SQLFluff has no diff mode (`fix --dry-run --diff` was removed in SQLFluff 3.x); `--dry-run` maps to the same read-only lint invocation as `--check`. The SQL dialect is never hardcoded; set it in the project config or forward `--dialect`.
- `internal/formattington/black.go` — Black (Python) integration (`.py` files; resolved from PATH or the bundled formatters; respects `[tool.black]` / `pyproject.toml`).
- MicroPython: no dedicated integration; the `micropython` language alias resolves to `black` in `cli.go` (MicroPython is a Python 3 subset and parses/formats cleanly under black; statement-level asm labels do not parse).
- `internal/formattington/shfmt.go` — shfmt integration.
- `internal/formattington/google_java_format.go` — google-java-format (Java) integration (`.java` files; bundled for macOS arm64, Linux arm64/x86_64, Windows x86_64; PATH-only on the rest). The `java` language alias resolves to `google-java-format` in `cli.go`. Whitespace-only diffs by default (`--cleanup=false` unless the user forwards their own `--cleanup`).
- `internal/formattington/dotnet_format.go` — dotnet format (C#/VB.NET) integration (`.cs`/`.vb` files; PATH-only; ships with the .NET SDK ≥ 6). The tool name is `dotnet-format`; the `csharp` and `vbnet` aliases resolve to `dotnet-format` in `cli.go`. Uses the `whitespace` subcommand with `--folder . --include <files>`; `--verify-no-changes` exit 2 is reported as `ExitFormatRequired`. Style/analyzers are opt-in via forwarded args (they restore/build project code).
- `internal/formattington/powershell.go` — PowerShell integration (`.ps1`, `.psm1`; PATH-only, no bundle). Lints via PSScriptAnalyzer (`Invoke-ScriptAnalyzer`) through `pwsh` (PowerShell 7+, falling back to `powershell.exe` on Windows); there is no in-place formatting, so check/dry-run/default all lint and findings exit 1. The `ps1` language alias resolves to `powershell` in `cli.go`.
- `internal/formattington/clang_format.go` — clang-format integration.
- `internal/formattington/rustfmt.go` — rustfmt (Rust) integration (`.rs` files; PATH-only; rustup component). The `rust` language alias resolves to `rustfmt` in `cli.go`. The edition is detected per file from the nearest Cargo.toml (`[package] edition`, else nearest `[workspace.package] edition` for inherited members); files are grouped per edition before batching; a caller-provided `--edition` wins.
- `internal/formattington/php_cs_fixer.go` — PHP-CS-Fixer (PHP) integration (`.php` files; PATH-only). The `php` language alias resolves to `php-cs-fixer` in `cli.go`. Native exit 8 (files need fixing) is reported as `ExitFormatRequired`; all other codes pass through.
- `internal/formattington/gofmt.go` and `internal/formattington/goimports.go` — Go integrations (`.go` files; PATH-only, no bundle). The `go` language alias resolves to `gofmt` in `cli.go`.
- `internal/formattington/ktfmt.go` — ktfmt (Kotlin) integration (`.kt`/`.kts` files; bundled for Linux x86_64, macOS arm64, Windows x86_64; PATH-only on the rest). The `kotlin` language alias resolves to `ktfmt` in `cli.go`. No style flag is ever injected.
- `internal/formattington/stylua.go` — StyLua (Lua) integration (`.lua` files; resolved from PATH or the bundled formatters; respects `stylua.toml` / `.stylua.toml`). The `lua` language alias resolves to `stylua` in `cli.go`.
- `internal/formattington/swift_format.go` — swift-format (Swift) integration (`.swift` files; PATH-only; ships with Swift 6+ toolchains — standalone `swift-format` tried first, falling back to `swift format` with the `format` prefix). The `swift` language alias resolves to `swift-format` in `cli.go`.
- `internal/formattington/rubocop.go` — RuboCop (Ruby) integration (`.rb`/`.rake`/`.gemspec`/`.ru` files; PATH-only). The `ruby` language alias resolves to `rubocop` in `cli.go`. A linter with layout cops — every mode lints (no writes); `-x` (--fix-layout) is passed verbatim by the user for the format-like path.
- `internal/formattington/dart_format.go` — dart format (Dart) integration (`.dart` files; PATH-only; ships with the Dart SDK). The tool name is `dart-format` (the binary is `dart`; invoked as `dart format`); the `dart` alias resolves to `dart-format` in `cli.go`. Check is `dart format --set-exit-if-changed -o none`.
- `internal/formattington/styler.go` — styler (R) integration (`.R`/`.r` files; PATH-only; needs `Rscript` + the styler package). The `r` language alias resolves to `styler` in `cli.go`. The R expression is built from the file list; forwarded flags are a usage error; check uses `dry = "fail"`.
- `internal/formattington/nixfmt.go` — nixfmt (Nix) integration (`.nix` files; PATH-only; no bundleable release matrix upstream). The `nix` language alias resolves to `nixfmt` in `cli.go`.
- `internal/formattington/gdformat.go` — gdformat (GDScript) integration (`.gd` files; PATH-only; Godot 4 / gdtoolkit 4.x). The `gdscript` language alias resolves to `gdformat` in `cli.go`.
- `internal/formattington/npm_groovy_lint.go` — npm-groovy-lint (Groovy) integration (`.groovy`/`.gradle` files; PATH-only; needs Node >= 18 and Java 11+; layout-only). The `groovy` language alias resolves to `npm-groovy-lint` in `cli.go`. Check mode is linter-flavored (`--failon error`), not a byte comparison; the `format` preset must be enabled in `.groovylintrc`.
- `internal/formattington/perltidy.go` — perltidy (Perl) integration (`.pl`, `.pm`, `.t` files; PATH-only; GPL-2.0+, never embedded). The `perl` language alias resolves to `perltidy` in `cli.go`. Check mode uses `perltidy --assert-tidy`; the default suppresses `.bak` backups with `-bext=/`.
- `internal/formattington/zig_fmt.go` — zig fmt (Zig) integration (`.zig` files only; PATH-only; ships with the Zig toolchain). The tool name is `zig-fmt` (the binary is `zig`; invoked as `zig fmt`); the `zig` language alias resolves to `zig-fmt` in `cli.go`. `.zon` is opt-in only (`--zon`, forwarded verbatim).
- `internal/formattington/mix_format.go` — mix format (Elixir) integration (`.ex`/`.exs` files; PATH-only; ships with Elixir/OTP). The tool name is `mix-format` (the binary is `mix`); the `elixir` language alias resolves to `mix-format` in `cli.go`. `--no-compile` is passed by default; check is `--check-formatted`. `.formatter.exs` is EXECUTED CODE discovered from the cwd.
- `internal/formattington/fourmolu.go` — fourmolu (Haskell) integration (`.hs` files; bundled for macOS arm64, macOS x86_64, Linux arm64, Linux x86_64; Windows stays PATH-only; Linux binaries are glibc-dynamic). The `haskell` language alias resolves to `fourmolu` in `cli.go`. `--mode check` exit 100 is reported as `ExitFormatRequired`; the diff is printed to stderr; all other codes pass through.
- `internal/formattington/scalafmt.go` — scalafmt (Scala) integration (`.scala`/`.sbt`/`.sc` files; PATH-only — the upstream natives fail hard on any `.scalafmt.conf` whose `version =` differs from the compiled-in one, so no bundle is shipped). The `scala` language alias resolves to `scalafmt` in `cli.go`.
- `internal/formattington/fprettify.go` — fprettify (Fortran) integration (free-form `.f90`/`.f95`/`.f03`/`.f08`/`.f18` files; PATH-only; GPL-3.0+, never embedded). The `fortran` language alias resolves to `fprettify` in `cli.go`. No native check mode; it is synthesized from `fprettify --diff` output.
- `internal/formattington/erlfmt.go` — erlfmt (Erlang) integration (`.erl`, `.hrl` files; PATH-only; needs Erlang/OTP 21+). The `erlang` language alias resolves to `erlfmt` in `cli.go`. No diff mode; dry-run maps to `-c`.
- `internal/formattington/jlfmt.go` — jlfmt (Julia) integration (`.jl` files; PATH-only; needs Julia >= 1.12 with the JuliaFormatter.jl app installed via `Pkg.Apps.add`). The `julia` language alias resolves to `jlfmt` in `cli.go`. Check mode uses `jlfmt --check` (exit 1 ⇒ `ExitFormatRequired`, 2 passes through); `--diff` needs git on PATH and degrades to `--check` with a note otherwise.
- `internal/formattington/cljfmt.go` — cljfmt (Clojure) integration (`.clj`/`.cljs`/`.cljc`/`.edn` files; bundled for macOS arm64, Linux arm64, Linux x86_64 static, Windows x86_64; EPL-1.0, unmodified upstream binaries). The `clojure` language alias resolves to `cljfmt` in `cli.go`. Subcommand-shaped (`cljfmt fix`/`cljfmt check`); `check` exits 1 on unformatted files, 2 on error.
- `internal/formattington/gleam_format.go` — gleam format (Gleam) integration (`.gleam` files; PATH-only; ships with the Gleam toolchain). The tool name is `gleam-format` (the binary is `gleam`; invoked as `gleam format`); the `gleam` language alias resolves to `gleam-format` in `cli.go`. `--check` exit non-zero (Error::Format = 1) is reported as `ExitFormatRequired`.
- `internal/formattington/gnatformat.go` — GNATformat (Ada) integration (`.ads`/`.adb` files; PATH-only; ships with GNAT Pro or Alire `alr install gnatformat`). The `ada` language alias resolves to `gnatformat` in `cli.go`. When no `.gpr` exists in the cwd, `--no-project` is passed; otherwise the tool auto-loads the single `.gpr`. `--check` exits 1 when files need formatting.
- `internal/formattington/fantomas.go` — Fantomas (F#) integration (`.fs`/`.fsi`/`.fsx` files; PATH-only — ships only as a .NET global tool). The `fsharp` language alias resolves to `fantomas` in `cli.go`. `--check` exit 99 is reported as `ExitFormatRequired`; 1 (error) passes through.
- `internal/formattington/ocamlformat.go` — ocamlformat (OCaml) integration (`.ml`/`.mli` files; PATH-only — no native binaries exist; built through opam). The `ocaml` language alias resolves to `ocamlformat` in `cli.go`. `--check` exit 1 is reported as `ExitFormatRequired` (also used for errors; stderr disambiguates).
- `internal/formattington/forge_fmt.go` — forge fmt (Solidity) integration (`.sol` files; PATH-only; ships with Foundry — foundryup, Homebrew, npm, release tarballs). The tool name is `forge-fmt` (the binary is `forge`; invoked as `forge fmt`); the `solidity` language alias resolves to `forge-fmt` in `cli.go`. `--check` exits 1 when changes are required (0 clean, diff printed).
- `internal/formattington/mojo_format.go` — mojo format (Mojo) integration (`.mojo` files; PATH-only; Linux glibc ≥ 2.34 / macOS arm64 only; Windows via WSL). The tool name is `mojo-format` (the binary is `mojo`; invoked as `mojo format`); the `mojo` language alias resolves to `mojo-format` in `cli.go`. No upstream check mode exists; check/dry-run are synthesized by formatting a temp copy and byte-comparing against the original (the user tree is never modified).
- `internal/formattington/odinfmt.go` — odinfmt (Odin) integration (`.odin` files; PATH-only; ships only via Homebrew/nixpkgs or built from the ols source; no Windows binary). The `odin` language alias resolves to `odinfmt` in `cli.go`. No upstream check mode exists; check/dry-run are synthesized by byte-comparing `odinfmt <file>` stdout against the original (tree untouched; one process per file). `-w` creates an `<file>_bk` backup first — documented.
- `internal/formattington/elm_format.go` — elm-format (Elm) integration (`.elm` files; bundled for Linux x86_64/arm64, macOS x86_64/arm64, Windows x86_64). The `elm` language alias resolves to `elm-format` in `cli.go`. Zero-configuration (gofmt philosophy); `--validate` exits 1 when files need formatting.
- `internal/formattington/markdownlint.go` — markdownlint integration (`.md`, `.markdown`; resolved from PATH or the bundled formatters; respects the tool's native configuration). Runs `markdownlint-cli` verbatim — the tool never rewrites files without an explicit `--fix`, so check/dry-run/default all lint and findings exit 1. There is no language alias; invoke it as `markdownlint` (also valid in `.formattington.json` commands).
- `internal/formattington/taplo.go` — Taplo (TOML) integration (`.toml` files; resolved from PATH or the bundled formatters; respects `taplo.toml`). The `toml` language alias resolves to `taplo` in `cli.go`.

`cli.go` is the single canonical entry point and the sole tool registry.

Markdown formatting is provided by Prettier via the `md`/`markdown` aliases; Markdown linting is provided by `markdownlint` (an exact tool name, no alias), which runs `markdownlint-cli` against `.md`/`.markdown` files. It never rewrites files without an explicit `--fix`, and its arguments are forwarded verbatim, so `--fix`, `--config`, and glob patterns work natively; configuration is discovered by the tool itself (for example `.markdownlint.json`). In `all`, `markdownlint` runs like every other registered tool (registered in survey order, near the end); findings exit 1, as with the other linters.

Before modifying a file, inspect its current implementation and callers. Keep responsibilities separated and avoid duplicating tool-specific logic across the orchestration layer.

New integrations should follow established patterns where appropriate, while allowing for differences in tool capabilities and behavior.

The orchestration layer owns tool selection, file discovery, execution strategy, and common reporting. Individual integrations own tool-specific invocation details and behavior.

Do not introduce unnecessary abstractions or a generalized plugin architecture without a demonstrated need.

## Presentation Order

User-facing tool and language enumerations default to **Stack Overflow Developer Survey 2026, "Programming, scripting, and markup languages", All Respondents, n = 14,099** (<https://survey.stackoverflow.co/2026/technology/data/languages>), licensed ODbL 1.0 (attribute it, as done here, whenever the ranking is published or repeated). File and folder names, testdata paths, and the bundled-formatter directory layout stay alphabetical — they are lookup paths, not narratives.

Ranked languages are mapped onto the tools that cover them; a tool takes the best rank of any language it covers. Equal-rank ties follow the survey's respondent counts, then the survey table order (Erlang 218 > Julia 211; Ada 172 = F# 172 with Ada first per survey order; Gleam 176 > Ada 172). `gofmt` precedes `goimports` (existing convention — goimports is the superset). Tools with no survey row (`markdownlint`, `taplo` — TOML/YAML/JSON/Markdown have no survey rank) sort **after all ranked tools, alphabetically among themselves**. `.formattington.json` (this repository's own config, generated by `formattington init` and validated by the test suite), the embedded catalogue `internal/formattington/init.json`, and every hardcoded order list in the docs mirror the registry exactly.

Everything Formattington *presents* to the user (help output, `all`/`--series` execution and exit-code aggregation order, `list` output, README tables and lists, this file's tool lists) follows this order. Re-derive the order on each new survey release — do not re-rank by downloads, stars, or intuition; the survey is the single source of truth.

## Development Environment

This is a Go project with no third-party dependencies.

Typical development commands:

```bash
make build
make test
make check
```

`make build` writes the binary to `./bin/formattington`; `make test` runs `go test ./...`; `make check` runs `go vet ./...`, verifies `gofmt` has nothing to change, and runs the test suite.

`make dist` (equivalently `./release.bash --build-only`) fetches the bundled formatters, cross-compiles the lite and full release binaries into `dist/`, smoke-tests the host's full binary, and packages the per-platform `formattington-<platform>-lite` / `formattington-<platform>-full` archives; `./release.bash` additionally verifies the version against versioner, tags, pushes, and creates the GitHub release with those archives attached.

Go 1.24 or newer is required (`go.mod` declares the language version).

When adding dependencies, prefer the Go standard library; a new module requires updating `go.mod` and `go.sum` with `go get` / `go mod tidy`.

## Implementation Guidelines

### Go

- Follow the existing code style and project conventions (`gofmt` is enforced by `make fmt-check`).
- Use `path/filepath` for filesystem paths and `os`/`io/fs` for filesystem access.
- Use `os/exec` for external commands.
- Pass subprocess arguments as a slice rather than constructing shell command strings.
- Avoid `sh -c` unless shell semantics are explicitly required.
- Handle missing executables, failed subprocesses, and invalid arguments clearly.
- Preserve useful stdout, stderr, and exit codes.
- Return errors instead of panicking; the CLI turns a missing tool into exit 127 and a usage error into exit 2.
- Keep exported identifiers documented; unexported helpers carry a short comment explaining their contract.

### File Discovery

- Match files by supported language and tool mappings.
- Respect project ignore rules where appropriate.
- Avoid processing generated, vendored, minified, or dependency files unintentionally.
- Handle spaces and special characters in file paths safely.
- Avoid passing an excessive number of arguments to a single process; account for operating-system command-line limits.
- Avoid discovering or processing the same file unnecessarily when multiple integrations are involved.

### Subprocesses and Performance

- Batch files for a tool when the tool supports multiple input paths.
- Avoid launching a separate process for every file unnecessarily.
- Keep independent tool execution efficient.
- Support sequential execution when requested.
- Do not introduce concurrency or caching complexity without a clear need.
- Measure performance changes against realistic mixed-language projects.

### Cross-Platform Compatibility

- Do not assume Unix-only path syntax or shell behavior.
- Use `path/filepath` and platform-aware process invocation.
- Account for differences in executable names and discovery on Windows.
- Avoid relying on a particular user's machine, absolute paths, or globally installed tools.
- Keep platform-specific behavior isolated when possible.

## Configuration and External Tools

Formattington orchestrates external tools; each tool retains responsibility for its own rules and specialized functionality.

- Respect native configuration files found in the relevant project context.
- Do not silently override a project's established configuration.
- Do not assume all tools are installed.
- Report missing tools clearly.
- Keep tool-specific arguments and behavior within the corresponding integration where practical.
- Allow users to choose between compatible tools that support the same file types.
- Avoid automatically running multiple conflicting tools against the same files.
- Do not impose a universal configuration format when native configurations are sufficient.
- Do not bundle additional binaries without updating the version table and README.

Examples of native configuration files include:

- `.prettierrc`
- `.clang-format`
- `.markdownlint.json`
- `pyproject.toml`

Bundled formatter binaries are not committed. `release.bash` fetches them at release time with `formatters/<tool>/update.sh` and the `formattington-<platform>-full` binaries embed them (`go:embed`, build tag `full`); on first use they are unpacked into the per-user cache directory. The `formattington-<platform>-lite` binaries embed none. `tools/` only holds the bundled build/helper tools (versioner, logamuffin, runner-spawner), which are pinned and tracked. `FORMATTINGTON_FORMATTERS_DIR` overrides the formatter lookup; `FindTool()` prefers bundled (or embedded) copies, then falls back to `PATH`.

## Testing and Validation

Run the test suite with:

```bash
make test
```

That is a thin wrapper around `go test ./...`. Tests live next to the code in `internal/formattington/` and use the standard `testing` package (no third-party test dependencies).

The suite runs without network access or installed formatter binaries:

- Unit tests point tool discovery at empty temporary trees and stub `PATH`.
- Integration tests use fake executable scripts in temporary directories.

Before considering a change complete:

1. Run `make check` using the project's configured workflow.
2. Verify CLI argument parsing and exit behavior.
3. Test paths containing spaces where filesystem or subprocess behavior changes.
4. Test missing executables and tool failures when relevant.
5. Check that native tool arguments are forwarded correctly.
6. Validate sequential and default execution behavior when orchestration changes.
7. Verify non-modifying operations do not modify files.
8. Test automatic fixes against representative inputs when adding fix support.
9. Avoid modifying unrelated files during formatting and linting tests.

Use the project's existing linting, formatting, and test tools where available. Do not invent commands for tools that are not configured.

## Scope and Change Discipline

- Keep changes focused on the requested task.
- Avoid unrelated refactors.
- Do not add dependencies without justification.
- Update documentation when public CLI behavior changes.
- Keep README examples consistent with the actual implementation.
- Do not claim a feature or operation is supported unless it has been implemented and validated.
- When behavior is ambiguous, inspect the existing code and tests before changing the interface.
- Prefer extending existing integrations over introducing parallel implementations of the same functionality.
- Preserve existing behavior unless a change explicitly requires otherwise.

## Definition of Done

A change is complete when it:

- Implements the requested behavior.
- Preserves the established CLI contract unless intentionally changed.
- Respects the capabilities and native behavior of the underlying tools.
- Handles errors and subprocess execution appropriately.
- Remains compatible with the project's supported platforms.
- Includes or updates tests where appropriate.
- Updates relevant documentation.

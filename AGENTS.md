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

The established public interface supports three selection modes.

### All configured tools

```bash
formattington all
formattington all --series
```

- `all` runs all configured formatter integrations against their matching files.
- By default, formatters run concurrently via goroutines.
- `--series` (before or after `all`, e.g. `formattington all --series`) requests sequential execution in registration order.
- Exit code is the last non-zero code in `shfmt`, `clang-format`, `black`, `prettier` order (missing tool = 127); `0` only if all succeed.
- A missing or failing formatter never prevents the others from running.

Until explicitly changed, `all` remains formatter-focused. Do not automatically introduce linting, fixing, or other operations into its execution.

### Specific tool

```bash
formattington prettier
formattington clang-format
formattington shfmt
```

Arguments following a tool selection should be forwarded to that tool where supported.

Examples:

```bash
formattington prettier --check
formattington clang-format --dry-run
formattington shfmt -d
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

All documented names above resolve: `c`/`cpp`/`c++` → `clang-format`, `bash` → `shfmt`, `python`/`py` → `black`, `md`/`markdown`/`html` → `prettier`. Prettier is the sole owner of Markdown formatting: `md`/`markdown` reuse the `prettier` dispatch and its `.prettierrc` handling, with no separate Markdown module. Unrecognized words produce a usage error (exit 2); an alias whose backend is not registered yet reports `'<name>' is not yet supported` (exit 2) rather than silently succeeding.

When multiple tools support the same file type, selection should respect configured tool choices rather than arbitrarily invoking conflicting tools.

When modifying CLI behavior, preserve these public command patterns unless the change explicitly updates the interface.

## Architecture

The program lives in `cmd/formattington/` (entry point) and `internal/formattington/` (the whole implementation).

Important files include:

- `cmd/formattington/main.go` — process entry point; calls `formattington.Main(os.Args[1:])` and exits with its return code.
- `internal/formattington/cli.go` — canonical command-line entry point and orchestration (sole `registry` of tools; `languageAliases` map).
- `internal/formattington/tools.go` — shared tool discovery (`FindTool`, `FormattersDir`), platform detection (`PlatformDir`), and subprocess helpers (`runTool`).
- `internal/formattington/embedded.go` — unpacks the formatter executables embedded in the "full" release binary into the per-user cache directory on first use.
- `internal/formattington/embedded_full.go` / `internal/formattington/embedded_lite.go` — the `go:embed` payload (build tag `full`) and its lite-build stub.
- `internal/formattington/discovery.go` — recursive file discovery (`DiscoverFiles`).
- `internal/formattington/config.go` — `.formattington.json` loading and validation.
- `internal/formattington/help.go` — usage text and the help output.
- `internal/formattington/version.go` — dynamic version resolution (versioner, `FORMATTINGTON_VERSION`).
- `internal/formattington/formatter.go` — the `Formatter` and `Options` types shared by the integrations.
- `internal/formattington/shfmt.go` — shfmt integration.
- `internal/formattington/clang_format.go` — clang-format integration.
- `internal/formattington/black.go` — Black (Python) integration (`.py` files; resolved from PATH or the bundled formatters; respects `[tool.black]` / `pyproject.toml`).
- `internal/formattington/prettier.go` — Prettier integration (`.js`, `.ts`, `.tsx`, `.jsx`, `.json`, `.html`, `.css`, `.scss`, `.yaml`, `.yml`, `.md`, `.markdown`; resolved from PATH or the bundled formatters; respects `.prettierrc`). Prettier is the sole owner of Markdown formatting; `md`/`markdown` are entries in `languageAliases` for `prettier`, and there is no separate Markdown module.
- `internal/formattington/powershell.go` — PowerShell integration (`.ps1`, `.psm1`; PATH-only, no bundle). Lints via PSScriptAnalyzer (`Invoke-ScriptAnalyzer`) through `pwsh` (PowerShell 7+, falling back to `powershell.exe` on Windows); there is no in-place formatting, so check/dry-run/default all lint and findings exit 1. The `ps1` language alias resolves to `powershell` in `cli.go`.
- `internal/formattington/taplo.go` — Taplo (TOML) integration (`.toml` files; resolved from PATH or the bundled formatters; respects `taplo.toml`). The `toml` language alias resolves to `taplo` in `cli.go`.
- `internal/formattington/gofmt.go` and `internal/formattington/goimports.go` — Go integrations (`.go` files; PATH-only, no bundle). The `go` language alias resolves to `gofmt` in `cli.go`.

`cli.go` is the single canonical entry point and the sole tool registry.

Markdown formatting is provided by Prettier via the `md`/`markdown` aliases; Markdown linting (for example `markdownlint`) is a separate operation outside this scope and is not yet integrated.

Before modifying a file, inspect its current implementation and callers. Keep responsibilities separated and avoid duplicating tool-specific logic across the orchestration layer.

New integrations should follow established patterns where appropriate, while allowing for differences in tool capabilities and behavior.

The orchestration layer owns tool selection, file discovery, execution strategy, and common reporting. Individual integrations own tool-specific invocation details and behavior.

Do not introduce unnecessary abstractions or a generalized plugin architecture without a demonstrated need.

## Development Environment

This is a Go project with no third-party dependencies.

Typical development commands:

```bash
make build
make test
make check
```

`make build` writes the binary to `./bin/formattington`; `make test` runs `go test ./...`; `make check` runs `go vet ./...`, verifies `gofmt` has nothing to change, and runs the test suite.

`make dist` (equivalently `./release.bash --build-only`) fetches the bundled formatters, cross-compiles the lite and full release binaries into `dist/`, smoke-tests the host's full binary, and packages the per-platform `formattington-lite-<platform>` / `formattington-full-<platform>` archives; `./release.bash` additionally verifies the version against versioner, tags, pushes, and creates the GitHub release with those archives attached.

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

Bundled formatter binaries are not committed. `release.bash` fetches them at release time with `formatters/<tool>/update.sh` and the `formattington-full-<platform>` binaries embed them (`go:embed`, build tag `full`); on first use they are unpacked into the per-user cache directory. The `formattington-lite-<platform>` binaries embed none. `tools/` only holds the bundled build/helper tools (versioner, logamuffin, runner-spawner), which are pinned and tracked. `FORMATTINGTON_FORMATTERS_DIR` overrides the formatter lookup; `FindTool()` prefers bundled (or embedded) copies, then falls back to `PATH`.

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

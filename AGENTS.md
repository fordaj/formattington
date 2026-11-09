# AGENTS.md

## Project Overview

Formattington is a cross-platform, multi-language formatting and code-quality tool orchestrator. It provides a unified command-line interface for invoking existing tools across multidisciplinary projects.

The goal is to bring project formatting and automated quality checks under one interface without reimplementing the underlying tools.

Formattington is written in Python and managed with `uv`.

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
- By default, formatters run concurrently via `concurrent.futures.ThreadPoolExecutor`.
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

The package currently lives in `src/formattington/`.

Important modules include:

- `cli.py` — canonical command-line entry point and orchestration (sole `FORMATTERS` registry; `pyproject.toml` declares `formattington.cli:main`).
- `__main__.py` — `python -m formattington` entry point (re-exports `cli.main`).
- `_tools.py` — shared tool discovery (`find_tool`) and subprocess helpers.
- `shfmt.py` — shfmt integration.
- `clang_format.py` — clang-format integration.
- `format_python.py` — Black (Python) integration (`.py` files; PATH-only, no bundle; respects `[tool.black]` / `pyproject.toml`).
- `prettier.py` — Prettier integration (`.js`, `.ts`, `.tsx`, `.jsx`, `.json`, `.html`, `.css`, `.scss`, `.yaml`, `.yml`, `.md`, `.markdown`; PATH-only, no bundle; respects `.prettierrc`). Prettier is the sole owner of Markdown formatting; `md`/`markdown` are `LANGUAGE_ALIASES` for `prettier` in `cli.py`, and there is no `markdown_format.py`.
- `powershell.py` — PowerShell integration (`.ps1`, `.psm1`; PATH-only, no bundle). Lints via PSScriptAnalyzer (`Invoke-ScriptAnalyzer`) through `pwsh` (PowerShell 7+, falling back to `powershell.exe` on Windows); there is no in-place formatting, so check/dry-run/default all lint and findings exit 1. The `ps1` language alias resolves to `powershell` in `cli.py`.
- `taplo.py` — Taplo (TOML) integration (`.toml` files; bundled under `tools/taplo/<version>/<platform>/`; respects `taplo.toml`). The `toml` language alias resolves to `taplo` in `cli.py`.

`cli.py` is the single canonical entry point and the sole `FORMATTERS` registry.

The Bash formatter module (`format_bash.py`) is planned but not yet present. Do not reference it as an existing file. `format_python.py` and `prettier.py` exist and are not part of the planned set. Markdown formatting is provided by Prettier via the `md`/`markdown` aliases; Markdown linting (for example `markdownlint`) is a separate operation outside this scope and is not yet integrated.

Before modifying a module, inspect its current implementation and callers. Keep responsibilities separated and avoid duplicating tool-specific logic across the orchestration layer.

New integrations should follow established patterns where appropriate, while allowing for differences in tool capabilities and behavior.

The orchestration layer owns tool selection, file discovery, execution strategy, and common reporting. Individual integrations own tool-specific invocation details and behavior.

Do not introduce unnecessary abstractions or a generalized plugin architecture without a demonstrated need.

## Development Environment

This is a Python project managed with `uv`.

Use the project environment and declared dependencies rather than installing packages globally.

Typical development commands:

```bash
uv sync
uv run python -m formattington
uv run formattington
```

Both `uv run python -m formattington` (via `src/formattington/__main__.py`, which re-exports `cli.main`) and the `uv run formattington` console script work.

`uv.lock` is checked in for reproducible installs; `uv sync` installs exactly the locked versions.

Python 3.14 or newer is required (`requires-python = ">=3.14"` in `pyproject.toml`; `.python-version` pins 3.14).

When adding dependencies, update the project configuration and lockfile using `uv` rather than manually editing resolved dependency versions.

## Implementation Guidelines

### Python

- Follow the existing code style and project conventions.
- Use `pathlib.Path` for filesystem paths.
- Use `subprocess.run` or an appropriate subprocess API for external commands.
- Pass subprocess arguments as a sequence rather than constructing shell command strings.
- Convert `Path` instances to strings when required by subprocess APIs.
- Avoid `shell=True` unless shell semantics are explicitly required.
- Handle missing executables, failed subprocesses, and invalid arguments clearly.
- Preserve useful stdout, stderr, and exit codes.
- Use type hints where they improve clarity.
- Prefer the Python standard library unless an additional dependency provides clear value.

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
- Use `pathlib` and platform-aware process invocation.
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

Bundled formatter binaries under `tools/` are pinned and tracked by design (see `tools/README.md` and the README bundled-tools policy).

`find_tool()` prefers bundled copies, then falls back to `PATH`.

## Testing and Validation

Run the test suite with:

```bash
uv sync
uv run pytest
```

Tests live in `tests/` and are configured via `[tool.pytest.ini_options]` in `pyproject.toml`.

The suite runs without network access or installed formatter binaries:

- Unit tests mock `platform`, `shutil.which`, and `subprocess.run`.
- Integration tests use fake executable scripts in temporary directories.

Before considering a change complete:

1. Run `uv run pytest` using the project's configured workflow.
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
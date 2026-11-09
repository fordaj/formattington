# AGENTS.md

## Project Overview

Formattington is a cross-platform, multi-language formatting orchestrator. It provides a unified command-line interface for invoking existing formatters across multidisciplinary projects.

The goal is to bring all project formatting under one interface without reimplementing the formatters themselves.

Formattington is written in Python and managed with `uv`.

## Core Principles

- **Reuse existing tools.** Delegate formatting to established formatters rather than implementing language-specific formatting logic.
- **Provide a consistent interface.** Users should not need to remember every formatter's command-line syntax.
- **Support multidisciplinary projects.** A single repository may contain many languages and file types.
- **Keep orchestration lightweight.** Avoid unnecessary work, subprocess launches, and dependencies.
- **Preserve native configuration.** Respect formatter-specific configuration files and repository conventions.
- **Remain cross-platform.** Account for Windows, macOS, and Linux differences in paths, executable discovery, and subprocess behavior.
- **Make failures visible.** Preserve meaningful exit statuses and diagnostics from underlying formatters.

## Command-Line Interface

The intended public interface supports three selection modes.

### All formatters

```bash
formattington all
formattington all --series
```

- `all` runs all configured formatter integrations against their matching files.
- `--series` requests sequential execution rather than the default execution strategy.

### Specific formatter

```bash
formattington prettier
formattington clang-format
formattington shfmt
```

Arguments following a formatter selection should be forwarded to that formatter where supported.

Examples:

```bash
formattington prettier --check
formattington clang-format --dry-run
formattington shfmt -d
```

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

Language aliases should resolve to the appropriate formatter integration and matching file types.

When modifying CLI behavior, preserve these public command patterns unless the change explicitly updates the interface.

## Architecture

The package currently lives in `src/formattington/`.

Important modules include:

- `cli.py` — canonical command-line entry point and orchestration (sole `FORMATTERS` registry; `pyproject.toml` declares `formattington.cli:main`).
- `__main__.py` — `python -m formattington` entry point (re-exports `cli.main`).
- `_tools.py` — shared tool discovery (`find_tool`) and subprocess helpers.
- `shfmt.py` — shfmt integration.
- `clang_format.py` — clang-format integration.
- `format_python.py` — Black (Python) integration (`.py` files; `PATH`-only, no bundle; respects `[tool.black]`/`pyproject.toml`).
- `.prettierrc` — Prettier configuration.
- `.clang-format` — clang-format configuration.

`cli.py` is the single canonical entry point and the sole `FORMATTERS` registry. Prettier, Bash, and Markdown formatter modules (`prettier.py`, `format_bash.py`, `markdown_format.py`) are planned but not yet present; do not reference them as existing files. (`format_python.py` now exists and is not part of the planned set.)

Before modifying a module, inspect its current implementation and callers. Keep responsibilities separated and avoid duplicating formatter-specific logic across the orchestration layer.

New formatter integrations should follow the established patterns in existing modules.

## Development Environment

This is a Python project managed with `uv`.

Use the project environment and declared dependencies rather than installing packages globally.

Typical development commands:

```bash
uv sync
uv run python -m formattington
uv run formattington
```

Both `uv run python -m formattington` (via `src/formattington/__main__.py`,
which re-exports `cli.main`) and the `uv run formattington` console script
work. `uv.lock` is checked in for reproducible installs; `uv sync` installs
exactly the locked versions. Python 3.14 or newer is required
(`requires-python = ">=3.14"` in `pyproject.toml`; `.python-version` pins
3.14).

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

- Match files by supported language and formatter mappings.
- Respect project ignore rules where appropriate.
- Avoid formatting generated, vendored, minified, or dependency files unintentionally.
- Handle spaces and special characters in file paths safely.
- Avoid passing an excessive number of arguments to a single process; account for operating-system command-line limits.

### Subprocesses and Performance

- Batch files for a formatter when the formatter supports multiple input paths.
- Avoid launching a separate formatter process for every file unnecessarily.
- Keep independent formatter execution efficient.
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

Formattington orchestrates external formatters; each formatter retains responsibility for its own formatting rules.

- Respect formatter configuration files found in the relevant project context.
- Do not silently override a project's established formatting configuration.
- Do not assume all formatters are installed.
- Report missing tools clearly.
- Keep formatter-specific arguments and behavior within the corresponding integration where practical.
- Bundled formatter binaries under `tools/` are pinned and tracked by design (see `tools/README.md` and the README bundled-tools policy); `find_tool()` prefers bundled copies then falls back to `PATH`. Do not bundle additional binaries without updating the version table and README.

## Testing and Validation

Run the test suite with:

```bash
uv sync
uv run pytest
```

Tests live in `tests/` and are configured via `[tool.pytest.ini_options]` in `pyproject.toml`. The suite runs with no network and no formatter binaries installed: unit tests mock `platform`, `shutil.which`, and `subprocess.run`, and integration tests use fake executable scripts in temporary directories.

Before considering a change complete:

1. Run `uv run pytest` using the project's configured workflow.
2. Verify CLI argument parsing and exit behavior.
3. Test paths containing spaces where filesystem or subprocess behavior changes.
4. Test missing executables and formatter failures when relevant.
5. Check that formatter arguments are forwarded correctly.
6. Validate sequential and default execution behavior when orchestration changes.
7. Avoid modifying unrelated files during formatting tests.

Use the project's existing linting, formatting, and test tools where available. Do not invent commands for tools that are not configured.

## Scope and Change Discipline

- Keep changes focused on the requested task.
- Avoid unrelated refactors.
- Do not add dependencies without justification.
- Update documentation when public CLI behavior changes.
- Keep README examples consistent with the actual implementation.
- Do not claim a feature is supported unless it has been implemented and validated.
- When behavior is ambiguous, inspect the existing code and tests before changing the interface.

## Definition of Done

A change is complete when it:

- Implements the requested behavior.
- Preserves the established CLI contract unless intentionally changed.
- Handles errors and subprocess execution appropriately.
- Remains compatible with the project's supported platforms.
- Includes or updates tests where appropriate.
- Updates relevant documentation.

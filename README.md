# Formattington

**One formatter interface for your entire project.**

Formattington brings all your code formatters together under one roof. Python, C++, JavaScript, Bash, Markdown, and more — one command to keep your multidisciplinary project consistently formatted.

## Why Formattington?

Modern projects rarely use just one language. Each language brings its own formatter, configuration, command-line options, and installation requirements.

Formattington provides a common interface over these tools, so you can spend less time remembering formatter commands and more time building things.

- **One interface** — format an entire project with a single command.
- **Multi-language** — format mixed-language repositories without juggling individual tools.
- **Flexible selection** — run every formatter, choose a specific formatter, or target a language.
- **Sequential or parallel execution** — choose whether formatters run concurrently or in sequence.
- **Existing configurations** — reuse native formatter configuration files.
- **Cross-platform ambitions** — designed for projects spanning Windows, macOS, and Linux.

## Usage

### Format the entire project

```bash
formattington all
```

Run all configured formatters against their matching files. This is
the default strategy: formatters run concurrently (via a thread pool), so
independent tools overlap. A missing or failing formatter does not
prevent the others from running; its failure is reported on stderr and
reflected in the exit code (see [Exit codes](#exit-codes)). Output from
different formatters may interleave in this mode.

### Run formatters sequentially

```bash
formattington all --series
```

Run formatters one after another (in `shfmt`, `clang-format`, `black`,
`prettier` order) instead of concurrently, for ordered output. The flag
is also accepted before `all` (for consistency with `--check`/`--dry-run`):

```bash
formattington --series all
formattington --check --series all
```

`--check` and `--dry-run` keep their usual meaning (they go before
`all`): in check mode every formatter still reports, whether you use the
concurrent default, `--series`, or both flags together.

### Choose a specific formatter

```bash
formattington prettier
formattington clang-format
formattington shfmt
formattington black
```

Pass additional arguments through to the underlying formatter:

```bash
formattington prettier --check
formattington clang-format --dry-run
formattington shfmt -d
```

Arguments are forwarded to the selected formatter, subject to Formattington's own argument handling.

### Format by language or file type

```bash
formattington c
formattington cpp
formattington c++
formattington bash
formattington python
formattington md
formattington markdown
formattington html
```

Language selection lets you target the relevant files without having to remember which formatter handles each extension.

## Exit codes

Formattington preserves underlying formatter exit codes and uses a fixed scheme for its own failures:

| Code | Meaning                                                                 |
| ---- | ----------------------------------------------------------------------- |
| 0    | Success. Formatting applied (or nothing needed formatting).             |
| 1    | Formatting changes required in check mode.                              |
| 2    | Usage error (for example, unknown formatter, exclusive `--check` and `--dry-run` given together, or unexpected arguments parsed by Formattington). |
| 127  | Formatter executable not found (no bundled binary and nothing on `PATH`). |

Mode semantics (Formattington owns the mode flag before the formatter
name; arguments after the formatter are forwarded verbatim):

| Formatter    | `--check` (before formatter)               | `--dry-run` (before formatter)     | Default (no flag)          |
| ------------ | ------------------------------------------ | ---------------------------------- | -------------------------- |
| shfmt        | `shfmt -l` — list unformatted files; exit 1 if any listed, else 0 | `shfmt -d` — print diff, do not modify | `shfmt -w` — write in place |
| clang-format | `clang-format --dry-run --Werror` — fail if any file needs formatting, do not modify | `clang-format --dry-run` — validate without modifying | `clang-format -i` — format in place (unless `-i`/`--in-place` already given) |
| black        | `black --check` — fail if any file needs formatting, do not modify | `black --diff` — show diff, do not modify | `black` in place (no flag) |
| prettier     | `prettier --check` — fail if any file needs formatting, do not modify | `prettier --list-different` — list files that differ, do not modify (Prettier has no `--diff`) | `prettier --write` — write in place (unless `--write`/`-w`, `--check`/`-c`, or `--list-different`/`-l` already given) |

When a formatter executable cannot be found, Formattington does not
raise a Python traceback. It prints a single line to stderr (for
example `formattington: Could not find 'shfmt'. Expected a bundled
executable under tools/shfmt/<version>/<platform>/shfmt or an
installation on PATH.`) and exits `127`. This happens identically in
default, `--check`, and `--dry-run` modes, for `shfmt`,
`clang-format`, `black`, and `prettier`, and under `all` (the remaining
formatters still run in both the concurrent default and `--series`).

Exit-code aggregation with `all`: the exit code is the **last non-zero
code in `shfmt`, `clang-format`, `black`, `prettier` order** (a missing
tool counts as `127` in its slot), or `0` only if every formatter
succeeded. The rule is deterministic regardless of completion order in
the concurrent default. For example, if `shfmt` returns `5` and
`prettier` returns `1`, `all` exits `1`; if `shfmt` returns `5` and the
rest succeed, `all` exits `5`. Use `--verbose` to see which executable
was resolved when a tool is present.

`formattington --help` and `formattington --version` do not require
any formatter executables and continue to work when tools are absent.

Install hints: `shfmt` comes from https://github.com/mvdan/sh ;
`clang-format` is part of LLVM (https://clang.llvm.org/docs/ClangFormat.html);
`black` is installed via pip (`pip install black` or
`uv tool install black`); `prettier` is installed via npm
(`npm install -g prettier`). Either a bundled copy under `tools/` (see
[Bundled tools](#bundled-tools)) or an installation on `PATH`
satisfies `find_tool()`; `black` and `prettier` are `PATH`-only (not bundled) and are
found via `PATH`.

## Supported formatters

Formattington integrates existing tools rather than reimplementing their formatting logic.

| Formatter                                                    | Status    | Typical use                                             |
| ------------------------------------------------------------ | --------- | ------------------------------------------------------- |
| [clang-format](https://clang.llvm.org/docs/ClangFormat.html) | Available | C, C++, and related languages                           |
| [shfmt](https://github.com/mvdan/sh)                         | Available | Shell scripts                                           |
| [Black](https://black.readthedocs.io/)                       | Available | Python (`.py`)                                          |
| [Prettier](https://prettier.io/)                             | Available | JavaScript, TypeScript, JSON, HTML, CSS, Markdown, YAML |
| Markdown formatting                                          | Planned   | Markdown-specific workflows                             |

Only `shfmt`, `clang-format`, `black`, `prettier`, `python` (alias for `black`), and
`all` are accepted by the CLI today. The Markdown-specific integration
is planned but not yet implemented — Formattington reports a usage
error (exit 2) for those planned names until they land. The `Usage` examples
above describe the intended public interface; the available subset is
`shfmt`, `clang-format`, `black`, and `prettier`. Note that Prettier
also formats `.md` files, so Markdown can be formatted via `prettier`
today; the planned Markdown-specific integration remains separate.

### Prerequisites

| Formatter    | Install method (bundled)                                                                 | Install method (PATH)                                                          | How Formattington finds it                                              |
| ------------ | -------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| shfmt        | Comes under `tools/shfmt/v3.14.1/<platform>/shfmt` in a fresh clone                    | `go install mvdan.cc/sh/v3/cmd/shfmt@latest` or your OS package manager        | Bundled first, then `PATH` via `find_tool("shfmt")`                     |
| clang-format | Comes under `tools/clang-format/v23.1.3/<platform>/clang-format` in a fresh clone       | Install LLVM/clang-tools for your OS (for example `brew install clang-format` or `apt install clang-format`) | Bundled first, then `PATH` via `find_tool("clang-format")`              |
| black        | Not bundled (PATH-only)                                                                | `pip install black` or `uv tool install black`                                | `PATH` via `find_tool("black")` (bundled lookup finds nothing)          |
| prettier     | Not bundled (PATH-only)                                                                | `npm install -g prettier` (or use `npx prettier` with the executable on `PATH`) | `PATH` via `find_tool("prettier")` (bundled lookup finds nothing)       |

Bundled versions are pinned in `tools/` for offline use; see
[Bundled tools](#bundled-tools) for the layout and refresh process.
If neither a bundle nor a `PATH` installation is found, the CLI prints
`formattington: Could not find ...` to stderr and exits 127 (see
[Exit codes](#exit-codes)). Formatter configuration files such as
`.prettierrc` and `.clang-format` remain in your repository and are
respected by the underlying tools.

## Installation

Formattington is a Python project managed with [uv](https://docs.astral.sh/uv/).

Clone the repository:

```bash
git clone https://github.com/fordaj/formattington.git
cd formattington
```

Install the project and its dependencies using the project's declared configuration:

```bash
uv sync
```

`uv.lock` is checked in for reproducible installs (`uv sync` installs
exactly the locked versions). The project requires Python 3.14 or
newer (`.python-version` pins 3.14 for development; `requires-python`
is `>=3.14` in `pyproject.toml`).

Run Formattington from the project environment (the console script
and the module entry point are equivalent):

```bash
uv run formattington all
uv run python -m formattington all
uv run formattington --version
uv run python -m formattington --version
```

### Formatter dependencies

Formattington orchestrates external formatters; it does not automatically make every formatter available simply by installing the Python package. A fresh clone already contains the pinned bundled executables under `tools/`, so `uv sync` plus the clones above are sufficient for offline use. If you prefer a system installation, put the tool on `PATH` instead — `find_tool()` prefers bundled copies and falls back to `PATH`. `black` is `PATH`-only (installed via `pip install black` or `uv tool install black`) and is not bundled under `tools/`. `prettier` is `PATH`-only (installed via `npm install -g prettier`) and is not bundled under `tools/`. See the [prerequisites table](#prerequisites) for per-formatter options.

## How it works

Formattington separates the orchestration layer from the formatting tools themselves.

1. **Discover** matching files in the project.
2. **Select** formatters based on the requested language or formatter name.
3. **Execute** the corresponding tools with the appropriate arguments
   (concurrently by default with `all`; one after another with
   `all --series`).
4. **Report** the results through a common interface (last non-zero exit
   code in formatter order, `0` only if all succeed).

The goal is to keep the interface simple while letting established formatter tools do what they do best.

## Bundled tools

Formattington looks for bundled formatter executables under
`tools/<name>/v<version>/<platform>/`, where `<platform>` is the
hyphenated `<os>-<arch>` name returned by `get_platform_dir()` (for
example `linux-x86_64`, `macos-arm64`, `windows-x86_64`). Bundled
tools are preferred; when none is found, lookup falls back to `PATH`.
`black` and `prettier` are `PATH`-only and are not bundled under `tools/`; install `black`
via `pip install black` or `uv tool install black`, and install `prettier`
via `npm install -g prettier`, and place the executable on `PATH`.

Use `--verbose` before the formatter name to see which executable was
resolved, or set `FORMATTINGTON_VERBOSE=1`:

```bash
formattington --verbose shfmt
formattington --verbose --check shfmt
```

Migration notes: the shfmt linux bundles were previously stored under
underscored names (`linux_arm64`, `linux_x86_64`). They are now
hyphenated (`linux-arm64`, `linux-x86_64`) to match `get_platform_dir()`
and clang-format. `find_tool()` still accepts the legacy underscored
form as a fallback. Bundled versions are compared numerically, so
`v10` wins over `v9` and `v23.1.3` wins over `v22.1.8`. On Windows,
`PATH` lookup tries both `name` and `name.exe`.

### Binary policy

Formattington intentionally commits pinned formatter binaries under
`tools/` (option (a) in #11). Bundled binaries are the first lookup
location in `find_tool()` and the fallback is `PATH`, so deleting or
updating a bundle never breaks the fallback — a missing bundle simply
defers to `PATH` and, if that also fails, exits `127` with a
one-line `formattington: Could not find ...` message.

Pinned upstream versions (recorded in `tools/README.md`):

- `shfmt` `v3.14.1` from https://github.com/mvdan/sh
- `clang-format` `v23.1.3` (plus a retained legacy `v22.1.8` macOS
  bundle) from the LLVM project

Keep these exact versions in sync with the docs when refreshing.
To refresh, download the new upstream release for each platform,
extract with `tools/clang-format/unzip.sh` where applicable, preserve
the `tools/<name>/v<version>/<platform>/` layout, and verify with
`formattington --verbose shfmt` (or `FORMATTINGTON_VERBOSE=1`).
There is no automatic fetch on first run; this keeps offline use
reliable at the cost of a large clone (about 45 MB for `tools/`).
See `tools/README.md` for the full layout and version table.

Repository hygiene: `__pycache__/`, `*.pyc`, `.DS_Store`, `.logs/`,
`.venv/`, `.pytest_cache/`, and `*.egg-info/` are ignored via
`.gitignore` and must not be tracked. The previously committed
`__pycache__/*.pyc` and `tools/**/*.DS_Store` files were untracked
with `git rm --cached`; only the formatter binaries remain tracked
under `tools/`.

## Testing

Run the test suite with:

```bash
uv sync
uv run pytest
```

Tests live in `tests/` and cover CLI parsing, tool discovery, subprocess boundaries, and argument forwarding. They run without network access and without formatter binaries installed, using mocks and fake executable scripts in temporary directories.

## Project status

Formattington is under active development. The core command-line interface and formatter integrations are being established, with additional languages, distribution options, and workflow improvements planned.

## License

See [LICENSE](LICENSE) for licensing information.
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

Run all configured formatters against their matching files.

### Run formatters sequentially

```bash
formattington all --series
```

Run formatters one after another instead of concurrently.

### Choose a specific formatter

```bash
formattington prettier
formattington clang-format
formattington shfmt
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
formattington md
formattington markdown
formattington html
```

Language selection lets you target the relevant files without having to remember which formatter handles each extension.

## Supported formatters

Formattington integrates existing tools rather than reimplementing their formatting logic.

| Formatter                                                    | Typical use                                                       |
| ------------------------------------------------------------ | ----------------------------------------------------------------- |
| [Prettier](https://prettier.io/)                             | JavaScript, TypeScript, JSON, HTML, CSS, Markdown, YAML, and more |
| [clang-format](https://clang.llvm.org/docs/ClangFormat.html) | C, C++, and related languages                                     |
| [shfmt](https://github.com/mvdan/sh)                         | Shell scripts                                                     |
| [Black](https://black.readthedocs.io/)                       | Python                                                            |
| Markdown formatting                                          | Markdown-specific workflows                                       |

Available integrations and file-type mappings depend on the current implementation and installed tools.

## Installation

Formattington is a Python project managed with [uv](https://docs.astral.sh/uv/).

Clone the repository:

```bash
git clone https://github.com/<owner>/Formattington.git
cd Formattington
```

Install the project and its dependencies using the project's declared configuration:

```bash
uv sync
```

Run Formattington from the project environment:

```bash
uv run formattington all
```

The exact commands depend on the project's configured package entry point. If the console script is not yet registered, the application can instead be run through its Python module or entry-point script.

### Formatter dependencies

Formattington orchestrates external formatters; it does not automatically make every formatter available simply by installing the Python package.

Install the formatters you intend to use and ensure they are discoverable by Formattington. Formatter configuration files, such as `.prettierrc` and `.clang-format`, can remain in your repository.

## How it works

Formattington separates the orchestration layer from the formatting tools themselves.

1. **Discover** matching files in the project.
2. **Select** formatters based on the requested language or formatter name.
3. **Execute** the corresponding tools with the appropriate arguments.
4. **Report** the results through a common interface.

The goal is to keep the interface simple while letting established formatter tools do what they do best.

## Bundled tools

Formattington looks for bundled formatter executables under
`tools/<name>/v<version>/<platform>/`, where `<platform>` is the
hyphenated `<os>-<arch>` name returned by `get_platform_dir()` (for
example `linux-x86_64`, `macos-arm64`, `windows-x86_64`). Bundled
tools are preferred; when none is found, lookup falls back to `PATH`.

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
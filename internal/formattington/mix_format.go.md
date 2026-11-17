# mix format (`mix-format`)

```mermaid
flowchart LR
    Src("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>.ex<br/>.exs<br/>covered files") --> Reg("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>formattington mix-format<br/>cli.go registry entry</u>")
    Reg --> Tool("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>mix via mix format<br/>PATH only</u>")
    Tool --> Ops("<img src='barnacle:list-checks.svg' width='20' alt=''/><br/><u>Operations: Format, Check</u>")
    click Reg href "cli.go.md"
    click Tool href "tools.go.md"
    click Ops href "formatter.go.md"
    class Src goFile
    class Reg goFile
    class Ops checkNode
    classDef goFile fill:#8fd8ef,stroke:#0277a8,color:#08303d
    classDef checkNode fill:#9fe8c9,stroke:#0f766e,color:#052e22
```

mix format is Elixir's built-in formatter; it ships with the Elixir installation and covers `.ex` and `.exs` files. The tool name is `mix-format` — the bare binary is `mix`, invoked as `mix format`.

## At a glance

|                      |                                                                                                                                                 |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Executable           | `mix` via `mix format`                                                                                                                          |
| Resolution           | PATH-only (ships with Elixir; needs Erlang/OTP on `PATH`)                                                                                       |
| Extensions           | `.ex`, `.exs`                                                                                                                                   |
| Operations           | Format, Check                                                                                                                                   |
| Language aliases     | `elixir` -> `mix-format`                                                                                                                        |
| Native configuration | `.formatter.exs` (**executed code** — returns a keyword list of formatter inputs/plugins; discovered from the current directory, not walked up) |

## Behavior

- Default invocation: `mix format --no-compile <files...>` — format in place with formatter plugins skipped so formatting never compiles the project.
- `--check`: `mix format --check-formatted --no-compile <files...>` — exit non-zero when any file would change; files are not modified.
- `--dry-run`: same as `--check` (never writes; there is no diff mode).
- Forwarded arguments: verbatim — for example, drop `--no-compile` yourself (or set formatter plugins in `.formatter.exs`) when plugins are required.
- Exit codes: any non-zero exit from `--check-formatted` is reported as `ExitFormatRequired` (1); other codes pass through.

## Notes

- **`.formatter.exs` is EXECUTED CODE** (returns a keyword list: inputs, plugins, import_deps, ...), discovered from the CURRENT DIRECTORY — never run this against untrusted projects. "Check mode" is not a sandbox.
- The tool name is `mix-format`, not `mix`, to avoid colliding with the build tool's own command shape.
- Formatter output can change between Elixir versions.
- `--no-compile` is passed by default so formatting never compiles the project; pass `--no-compile`-free arguments yourself when plugins are needed.
- Apache-2.0; PATH-only, never redistributed by Formattington.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/elixir-lang/elixir

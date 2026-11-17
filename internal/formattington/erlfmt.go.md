# erlfmt (`erlfmt`)

```mermaid
flowchart LR
    Src("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>.erl<br/>.hrl<br/>covered files") --> Reg("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>formattington erlfmt<br/>cli.go registry entry</u>")
    Reg --> Tool("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>erlfmt<br/>PATH only</u>")
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

erlfmt (github.com/WhatsApp/erlfmt) is the mature Erlang formatter. Formattington registers it for Erlang source and header files.

## At a glance

|                      |                                                                                                             |
| -------------------- | ----------------------------------------------------------------------------------------------------------- |
| Executable           | `erlfmt`                                                                                                    |
| Resolution           | PATH-only (no bundleable binaries; ships as a rebar3 plugin or escript built from it; needs Erlang/OTP 21+) |
| Extensions           | `.erl`, `.hrl`                                                                                              |
| Operations           | Format, Check                                                                                               |
| Language aliases     | `erlang` → `erlfmt`                                                                                         |
| Native configuration | none (erlfmt takes no configuration file by design)                                                         |

## Behavior

- Default invocation: `erlfmt -w` — format in place
- `--check`: `erlfmt -c` — fail if any file needs formatting; files are not modified
- `--dry-run`: same as `--check` (there is no diff mode)
- Forwarded arguments: forwarded verbatim to `erlfmt`
- Exit codes: no remapping — `erlfmt -c` exits 1 when any file needs formatting (0 otherwise)

## Notes

- There are no bundleable binaries — erlfmt ships as source (a rebar3 plugin or an escript built from it), so it is PATH-only and needs a working Erlang/OTP 21+ install.
- Invoke the escript form (`erlfmt -w` / `erlfmt -c`); a standalone `erlfmt` escript or one made available via the project's rebar3 build both satisfy discovery.
- erlfmt takes no configuration file by design; formatting style is fixed by the tool.
- `-c` is the check mapping and the dry-run mapping; there is no diff mode, so dry-run does not print a diff.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/WhatsApp/erlfmt

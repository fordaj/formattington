# gofmt (`gofmt`)

```mermaid
flowchart LR
    Src("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>.go<br/>covered files") --> Reg("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>formattington gofmt<br/>cli.go registry entry</u>")
    Reg --> Tool("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>gofmt<br/>PATH only</u>")
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

gofmt is the canonical Go formatter and ships with the Go installation. It covers `.go` files.

## At a glance

|                      |                                                                |
| -------------------- | -------------------------------------------------------------- |
| Executable           | `gofmt`                                                        |
| Resolution           | PATH-only (part of the Go installation; `gofmt` comes with Go) |
| Extensions           | `.go`                                                          |
| Operations           | Format, Check                                                  |
| Language aliases     | `go` -> `gofmt`                                                |
| Native configuration | none (zero-configuration)                                      |

## Behavior

- Default invocation: `gofmt -w <files...>` — write the result to the source file instead of stdout (format in place).
- `--check`: `gofmt -l <files...>` — list files whose formatting differs; the output is captured and reported as `ExitFormatRequired` (1) when it is non-empty, otherwise 0.
- `--dry-run`: `gofmt -d <files...>` — display diffs instead of rewriting files.
- Forwarded arguments: verbatim (for example, extra flags after the file list or in place of it).
- Exit codes: `gofmt -l` always exits 0, so a non-empty `-l` listing is reported as `ExitFormatRequired` (1); other codes pass through.

## Notes

- `gofmt -l` exits 0 even when files need formatting, so Formattington captures the output and turns a non-empty listing into exit 1.
- `gofmt` and [goimports](goimports.go.md) cover the same `.go` files: `gofmt` is canonical formatting, `goimports` additionally fixes import grouping. `formattington go` resolves to `gofmt`; use `formattington goimports` when imports should also be fixed.
- `gofmt` and `goimports` are PATH-only and are never bundled; `list` probes them via `go version <path>`.
- BSD-3-Clause; PATH-only, never redistributed by Formattington.

## Related pages

- [goimports](goimports.go.md) — gofmt plus import fixing
- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/golang/go (cmd/gofmt)

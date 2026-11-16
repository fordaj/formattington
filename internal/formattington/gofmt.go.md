# gofmt (`gofmt`)

```mermaid
flowchart LR
    Src[".go<br/>covered files"] --> Reg["formattington gofmt<br/>cli.go registry entry"]
    Reg --> Tool["gofmt<br/>PATH only"]
    Tool --> Ops["Operations: Format, Check"]
    click Reg href "cli.go.md"
    click Tool href "tools.go.md"
    click Ops href "formatter.go.md"
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

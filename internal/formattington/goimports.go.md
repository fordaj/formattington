# goimports (`goimports`)

```mermaid
flowchart LR
    Src[".go<br/>covered files"] --> Reg["formattington goimports<br/>cli.go registry entry"]
    Reg --> Tool["goimports<br/>PATH only"]
    Tool --> Ops["Operations: Format, Check"]
    click Reg href "cli.go.md"
    click Tool href "tools.go.md"
    click Ops href "formatter.go.md"
```

goimports formats Go files like gofmt and additionally fixes import grouping: missing imports are added, unused imports are removed, and imports are grouped into sections. It covers `.go` files.

## At a glance

|                      |                                                                                                              |
| -------------------- | ------------------------------------------------------------------------------------------------------------ |
| Executable           | `goimports`                                                                                                  |
| Resolution           | PATH-only (`go install golang.org/x/tools/cmd/goimports@latest`; ensure `$(go env GOPATH)/bin` is on `PATH`) |
| Extensions           | `.go`                                                                                                        |
| Operations           | Format, Check                                                                                                |
| Language aliases     | none (invoke `goimports` directly; `go` resolves to `gofmt`)                                                 |
| Native configuration | none (individual import options such as `-local` are forwarded verbatim and not overridden)                  |

## Behavior

- Default invocation: `goimports -w <files...>` — write the result to the source file instead of stdout (format in place, fixing imports).
- `--check`: `goimports -l <files...>` — list files whose formatting differs; the output is captured and reported as `ExitFormatRequired` (1) when it is non-empty, otherwise 0.
- `--dry-run`: `goimports -d <files...>` — display diffs instead of rewriting files.
- Forwarded arguments: verbatim; individual import options (for example `-local`) are forwarded as given and are not overridden.
- Exit codes: `goimports -l` always exits 0, so a non-empty `-l` listing is reported as `ExitFormatRequired` (1); other codes pass through.

## Notes

- `goimports` is the superset of [gofmt](gofmt.go.md): it applies the same canonical formatting and also fixes imports. `gofmt` precedes `goimports` in `all` execution order by convention.
- Go has two formatters covering the same `.go` files; selection should respect configured tool choices rather than running both conflictingly — use `--series` with `all` to run `gofmt` then `goimports` sequentially, or list only one in a config file.
- `gofmt` and `goimports` are PATH-only and are never bundled; `list` probes them via `go version <path>`.
- BSD-3-Clause; PATH-only, never redistributed by Formattington.

## Related pages

- [gofmt](gofmt.go.md) — canonical Go formatting
- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/golang/tools (cmd/goimports)

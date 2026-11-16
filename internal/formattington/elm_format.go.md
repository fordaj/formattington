# elm-format (`elm-format`)

```mermaid
flowchart LR
    Src[".elm<br/>covered files"] --> Reg["formattington elm-format<br/>cli.go registry entry"]
    Reg --> Tool["elm-format<br/>PATH or bundled"]
    Tool --> Ops["Operations: Format, Check"]
    click Reg href "cli.go.md"
    click Tool href "tools.go.md"
    click Ops href "formatter.go.md"
```

[elm-format](https://github.com/avh4/elm-format) is the Elm formatter, built on the gofmt philosophy of zero configuration. Formattington registers it for `.elm` files.

## At a glance

|                      |                                                                                                                                                                                             |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Executable           | `elm-format`                                                                                                                                                                                |
| Resolution           | Resolved from PATH or the bundled formatters (bundled for Linux x86_64/arm64, macOS x86_64/arm64, Windows x86_64; no Windows arm64 asset exists upstream, so that platform stays PATH-only) |
| Extensions           | `.elm`                                                                                                                                                                                      |
| Operations           | Format, Check                                                                                                                                                                               |
| Language aliases     | `elm` → `elm-format`                                                                                                                                                                        |
| Native configuration | none (zero-configuration by design)                                                                                                                                                         |

## Behavior

- Default invocation: `elm-format --yes` — format in place, accepting all overwrite prompts (the interactive prompt is suppressed)
- `--check`: `elm-format --validate` — print a JSON report and exit 1 when files need formatting (0 when clean)
- `--dry-run`: same as `--check` (there is no diff mode)
- Forwarded arguments: forwarded verbatim (`--elm-version` is an example)
- Exit codes: no remapping — error paths also exit 1, the JSON report itself disambiguates

## Notes

- Zero-configuration by design — there is no config file, and none is invented.
- The full release archives bundle the five upstream static binaries (Linux x86_64/arm64, macOS x86_64/arm64, Windows x86_64). The mac-arm64 asset is ~42 MB extracted — an unstripped upstream build, redistributed unmodified.
- `--validate` exits 1 both when files need formatting and on error; the report (a JSON document) disambiguates.
- `--yes` suppresses the interactive overwrite prompt in default mode, so runs never block waiting for input.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/avh4/elm-format

# swift-format (`swift-format`)

```mermaid
flowchart LR
    Src("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>.swift<br/>covered files") --> Reg("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>formattington swift-format<br/>cli.go registry entry</u>")
    Reg --> Tool("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>swift-format<br/>PATH only</u>")
    Tool --> Ops("<img src='barnacle:list-checks.svg' width='20' alt=''/><br/><u>Operations: Format, Lint / Check</u>")
    click Reg href "cli.go.md"
    click Tool href "tools.go.md"
    click Ops href "formatter.go.md"
    class Src goFile
    class Reg goFile
    class Ops checkNode
    classDef goFile fill:#8fd8ef,stroke:#0277a8,color:#08303d
    classDef checkNode fill:#9fe8c9,stroke:#0f766e,color:#052e22
```

swift-format is the official Swift formatter (swiftlang/swift-format), covering `.swift` files. It ships with Swift 6+ toolchains (Xcode 16+) and is PATH-only.

## At a glance

|                      |                                                                                           |
| -------------------- | ----------------------------------------------------------------------------------------- |
| Executable           | `swift-format` (standalone, preferred) or `swift` via `swift format`                      |
| Resolution           | PATH-only (Swift 6+ toolchains; standalone tried first, then `swift format`)              |
| Extensions           | `.swift`                                                                                  |
| Operations           | Format, Lint / Check                                                                      |
| Language aliases     | `swift` -> `swift-format`                                                                 |
| Native configuration | `.swift-format` (JSON; discovered walking up from each file; `--configuration` overrides) |

## Behavior

- Default invocation: `swift format -i --ignore-unparsable-files` (or the standalone equivalent) — format in place.
- `--check`: `swift format lint --strict --ignore-unparsable-files` — exit 1 on findings; files are not modified.
- `--dry-run`: same as `--check` (no diff mode exists).
- Forwarded arguments: verbatim; `--configuration` overrides the discovered `.swift-format`.
- Exit codes: no remapping; check mode exits 1 on findings (via `lint --strict`).

## Notes

- Resolution prefers a standalone `swift-format` on `PATH` (newer standalone installs, e.g. from Homebrew); when absent, it falls back to `swift format`, the subcommand dispatcher included with Swift 6+ toolchains (Xcode 16+). Arguments are identical either way.
- `lint --strict` turns findings into errors (exit 1); without `--strict` findings are warnings and exit 0 — Formattington always uses `--strict` in check mode.
- `--ignore-unparsable-files` skips unparsable input instead of failing the whole batch.
- Configuration is a `.swift-format` JSON file discovered by walking up from each file; `--configuration` overrides it.
- There is no diff mode, so `--dry-run` maps to the same `--strict` lint invocation as `--check`.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/swiftlang/swift-format

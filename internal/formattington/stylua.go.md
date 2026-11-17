# StyLua (`stylua`)

```mermaid
flowchart LR
    Src("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>.lua<br/>covered files") --> Reg("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>formattington stylua<br/>cli.go registry entry</u>")
    Reg --> Tool("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>stylua<br/>bundled, then PATH</u>")
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

StyLua is the standard Lua formatter, covering `.lua` files. It is resolved from the bundled formatters (full release archives) or from `PATH`, and respects `stylua.toml` / `.stylua.toml`.

## At a glance

|                      |                                                                                                                                 |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Executable           | `stylua`                                                                                                                        |
| Resolution           | Bundled in the full release archives (no Windows arm64 asset upstream — that platform is PATH-only), then `PATH` via `FindTool` |
| Extensions           | `.lua`                                                                                                                          |
| Operations           | Format, Check                                                                                                                   |
| Language aliases     | `lua` -> `stylua`                                                                                                               |
| Native configuration | `stylua.toml` / `.stylua.toml` (or `.editorconfig` unless `--no-editorconfig` is passed)                                        |

## Behavior

- Default invocation: `stylua` — format in place.
- `--check`: `stylua --check` — fail if any file needs formatting; files are not modified.
- `--dry-run`: `stylua --check --output-format=unified` — print unified diffs, do not modify.
- Forwarded arguments: verbatim; configuration is discovered by the tool itself.
- Exit codes: no remapping; `--check` exits 1 natively when any file is not formatted.

## Notes

- `--output-format=unified` is only honored together with `--check`, which is why the dry-run mapping is `--check --output-format=unified`.
- StyLua respects `stylua.toml` / `.stylua.toml` (and `.editorconfig` unless `--no-editorconfig` is passed); configuration is discovered by the tool itself.
- Bundled via `formatters/stylua/update.sh` (`brew install stylua` / `cargo install stylua` for a system install); no Windows arm64 asset is published upstream, so that platform relies on `PATH`.
- `--check` compares a diff between all input files and exits 1 when any file is not formatted; input files are never overwritten in check mode.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/JohnnyMorganz/StyLua

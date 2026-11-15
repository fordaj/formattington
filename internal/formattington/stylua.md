# StyLua (`stylua`)

StyLua is the standard Lua formatter, covering `.lua` files. It is resolved from the bundled formatters (full release archives) or from `PATH`, and respects `stylua.toml` / `.stylua.toml`.

## At a glance

| | |
| --- | --- |
| Executable | `stylua` |
| Resolution | Bundled in the full release archives (no Windows arm64 asset upstream — that platform is PATH-only), then `PATH` via `FindTool` |
| Extensions | `.lua` |
| Operations | Format, Check |
| Language aliases | `lua` -> `stylua` |
| Native configuration | `stylua.toml` / `.stylua.toml` (or `.editorconfig` unless `--no-editorconfig` is passed) |

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

- [Command dispatch and the tool registry](cli.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.md)
- [Execution options and the Formatter contract](formatter.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/JohnnyMorganz/StyLua

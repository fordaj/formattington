# The logamuffin bundle (`tools/logamuffin/`)

[Logamuffin](https://github.com/fordaj/logamuffin) is the structured logger used by this repository's scripts: timestamped, leveled `log <level> <message>` records. The scripts' `log()` wrapper ([scripts/entry.md](../../scripts/entry.md)) forwards to the pinned binary when it is present and falls back to an equivalent `printf` implementation when it is not, so a fresh clone works without it.

The scripts themselves document the contract (`log INFO "..."`), the `LOG_LEVEL`/`LOGAMUFFIN_LEVEL` filter, and the `LOG_SOURCE` caller-context field that entry.sh injects.

## Bundled platforms

`v0.1.3.0.35/macos-arm64`, `macos-x86_64`, `linux-arm64`, `linux-x86_64`, `windows-arm64`, `windows-x86_64` (the linker picks the host's platform via `LOGAMUFFIN_BIN`).

## `update.sh`

Refreshes from `fordaj/logamuffin` releases: assets named `Logamuffin_<tag>_<os>-<arch>.tar.gz`/`.zip` (legacy `.7z` accepted) mapped onto the bundle layout, verified against the release's `checksums.txt` when present. Only the binary is kept. See [../README.md](../README.md) for the shared bundle contract (`--dry-run`, `--tag`, staged atomic swap).

## Related pages

- [../README.md](../README.md) — the bundle index
- [../../scripts/entry.md](../../scripts/entry.md) — the `log()` wrapper that calls it

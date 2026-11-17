# The logamuffin bundle (`tools/logamuffin/`)

```mermaid
flowchart LR
    Gh("<img src='barnacle:tag.svg' width='20' alt=''/><br/>fordaj/logamuffin<br/>GitHub release") --> Upd("<img src='barnacle:terminal.svg' width='20' alt=''/><br/>update.sh<br/>gh · checksums · stage · swap")
    Upd --> Dir("<img src='barnacle:folder.svg' width='20' alt=''/><br/>tools/logamuffin/v254.1.3.0.48/<br/>per-platform binary")
    Dir --> Log("<img src='barnacle:scroll-text.svg' width='20' alt=''/><br/><u>log() in scripts/entry.sh<br/>timestamped leveled records</u>")
    class Gh tagNode
    class Upd shFile
    class Dir folder
    classDef tagNode fill:#f6b8d4,stroke:#be185d,color:#500724
    classDef shFile fill:#abe07a,stroke:#3f6212,color:#16280a
    classDef folder fill:#f0c558,stroke:#8a6d1c,color:#332a05
    click Log href "../../scripts/entry.sh.md"
```

[Logamuffin](https://github.com/fordaj/logamuffin) is the structured logger used by this repository's scripts: timestamped, leveled `log <level> <message>` records. The scripts' `log()` wrapper ([scripts/entry.sh.md](../../scripts/entry.sh.md)) forwards to the pinned binary when it is present and falls back to an equivalent `printf` implementation when it is not, so a fresh clone works without it.

The scripts themselves document the contract (`log INFO "..."`), the `LOG_LEVEL`/`LOGAMUFFIN_LEVEL` filter, and the `LOG_SOURCE` caller-context field that entry.sh injects.

## Bundled platforms

`v254.1.3.0.48/macos-arm64`, `macos-x86_64`, `linux-arm64`, `linux-x86_64`, `windows-arm64`, `windows-x86_64` (the linker picks the host's platform via `LOGAMUFFIN_BIN`).

## `update.sh`

Refreshes from `fordaj/logamuffin` releases: assets named `Logamuffin_<tag>_<os>-<arch>.tar.gz`/`.zip` (legacy `.7z` accepted) mapped onto the bundle layout, verified against the release's `checksums.txt` when present. Only the binary is kept. See [../update.sh.md](../update.sh.md) for the shared refresh implementation (`--dry-run`, `--tag`, staged atomic swap) and [../README.md](../README.md) for the bundle contract.

## Related pages

- [../README.md](../README.md) — the bundle index
- [../../scripts/entry.sh.md](../../scripts/entry.sh.md) — the `log()` wrapper that calls it

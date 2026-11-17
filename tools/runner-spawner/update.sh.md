# The runner-spawner bundle (`tools/runner-spawner/`)

```mermaid
flowchart LR
    Gh("<img src='barnacle:tag.svg' width='20' alt=''/><br/>fordaj/runner-spawner<br/>GitHub release") --> Upd("<img src='barnacle:terminal.svg' width='20' alt=''/><br/>update.sh<br/>gh · checksums · stage · swap")
    Upd --> Dir("<img src='barnacle:folder.svg' width='20' alt=''/><br/>tools/runner-spawner/v0.0.1.2.6/<br/>per-platform binary")
    Dir --> CI("<img src='barnacle:server.svg' width='20' alt=''/><br/>self-hosted CI runners<br/>driven by .runner-spawner.json")
    class Gh tagNode
    class Upd shFile
    class Dir folder
    classDef tagNode fill:#f6b8d4,stroke:#be185d,color:#500724
    classDef shFile fill:#abe07a,stroke:#3f6212,color:#16280a
    classDef folder fill:#f0c558,stroke:#8a6d1c,color:#332a05
```

[Runner-spawner](https://github.com/fordaj/runner-spawner) manages self-hosted GitHub Actions runners. In this repository it drives the planned self-hosted CI on macOS and Windows runners (see [README.md — Continuous integration](../../README.md)); it operates on the committed `.runner-spawner.json`:

```json
{
  "owner": "fordaj",
  "repo": "formattington",
  "runnerNames": ["runner01"],
  "runnersDir": ".runners",
  "runnerGroup": "Default",
  "runnerVersion": "2.338.2"
}
```

The runner tree itself (`.runners/`) is machine-local state and is not committed.

## Bundled platforms

`v0.0.1.0.4/macos-arm64`, `macos-x86_64`, `linux-arm64`, `linux-x86_64`, `windows-arm64`, `windows-x86_64`. Note: runner-spawner reports unsupported on Linux (macOS + Windows only); the Linux binaries are shipped for uniform bundle layout, and the platform env files carry this caveat.

The pinned binary selection goes through the [env cascade](../../scripts/env.sh.md) (`RUNNER_SPAWNER_BIN`); `RUNNER_SPAWNER_REPO` is intentionally **not** the upstream — the scripts set `RUNNER_SPAWNER_TOOL_REPO=fordaj/runner-spawner` for the bundle refresh instead.

## `update.sh`

Refreshes from `fordaj/runner-spawner` releases: assets named `RunnerSpawner_<tag>_<os>-<arch>.tar.gz`/`.zip` (legacy `.7z` accepted) mapped onto the bundle layout. Verification is mandatory: a release with no `checksums.txt` aborts the refresh. Only the binary is kept. See [../update.sh.md](../update.sh.md) for the shared refresh implementation (`--dry-run`, `--tag`, staged atomic swap) and [../README.md](../README.md) for the bundle contract.

## Related pages

- [../README.md](../README.md) — the bundle index
- [../../scripts/env.sh.md](../../scripts/env.sh.md) — the cascade selecting `RUNNER_SPAWNER_BIN`

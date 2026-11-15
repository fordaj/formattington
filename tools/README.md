# Pinned helper tools (`tools/`)

`tools/` carries the repository's pinned helper binaries — build, release, logging, and docs tooling — plus one `update.bash` per tool that refreshes them. These are **not** the formatters Formattington orchestrates (those live in `formatters/` and are fetched at release time); they are the tools used to build, test, log, and document this repository itself.

## Bundling convention

Every bundle follows the same layout:

```text
tools/<name>/
  update.bash           # refresh from the upstream GitHub release
  v<version>/<platform>/<binary>[.exe]
```

- Binaries are pinned by version directory; the pinned version is named by `*_VERSION_DIR` in [`scripts/.common.env`](../scripts/env.md).
- `<platform>` is the hyphenated `<os>-<arch>` name (`linux-x86_64`, `macos-arm64`, `windows-x86_64`, …), matching `PlatformDir()`.
- `update.bash` downloads the latest (or `--tag`ged) upstream release via `gh`, verifies checksums, stages into a temporary directory, swaps in atomically, and deletes superseded version folders only after the new one is staged — a failed refresh leaves the previous bundle intact. Each script has `--dry-run`, `--help`, and resolves its own directory, so it can run from anywhere.
- Only the binary is kept, except [barnacle](barnacle/update.md), which also retains the upstream MIT `LICENSE` as `<tag>/LICENSE`.

## The bundles

| Tool | Used for | Docs |
| --- | --- | --- |
| [versioner](versioner/update.md) | Computes the repository version (tag) for builds and releases | [update.bash](versioner/update.md) |
| [logamuffin](logamuffin/update.md) | Structured logging backend for `log()` in the scripts | [update.bash](logamuffin/update.md) |
| [runner-spawner](runner-spawner/update.md) | Manages self-hosted CI runners (driven by `.runner-spawner.json`) | [update.bash](runner-spawner/update.md) |
| [barnacle](barnacle/update.md) | Renders this documentation site | [update.bash](barnacle/update.md) |

`scripts/.common.env` and `scripts/.<platform>.env` select these via `*_VERSION_DIR`/`*_BIN`; see [env.md](../scripts/env.md). Archives are platform-native (`.tar.gz` on macOS/Linux, `.zip` on Windows); there is no 7zip dependency anywhere in this repository.

## Refreshing a pinned tool

```bash
tools/barnacle/update.bash            # download the latest release, verify, swap
tools/barnacle/update.bash --dry-run  # show what would change
```

(Same flags for `versioner`, `logamuffin`, `runner-spawner`.) Refreshes are checksum-verified; All four verifiers accept `checksums.txt` or `SHA256SUMS.txt` (missing file: a warning, except runner-spawner, which refuses to install unverified binaries). Legacy `.7z` assets are still readable (a p7zip install is only needed until new releases ship platform-native archives).

## Related pages

- [scripts/env.md](../scripts/env.md) — how the cascade selects these binaries
- [scripts/docs.md](../scripts/docs.md) — barnacle in action
- [../release.md](../release.md) — the release pipeline
- [README.md — repository hygiene](../README.md)

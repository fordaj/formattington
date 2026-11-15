# Repository script tooling (`scripts/`)

The `scripts/` directory carries the repository's build/release/docs tooling: an environment cascade the other scripts source, plus one entry point per task. Everything here is bash; nothing requires a global install beyond what each script documents.

Vernacular used in the headers (`log INFO`, `$REPO_ROOT_DIR`, `VERSIONER_BIN`): read [entry.md](entry.md) first, then [env.md](env.md).

## The cascade (why the dotfiles exist)

- `scripts/entry.sh` — sources `scripts/env.sh` and defines `log()` (logamuffin, with a printf fallback). Sourced by everything else. See [entry.md](entry.md).
- `scripts/env.sh` — locates and sources the cascade below, then anchors the repository-relative `*_BIN` tool paths to absolute paths. See [env.md](env.md).
- `scripts/.common.env`, `scripts/.<os>-<arch>.env` — the cascade proper. Hidden files, so barnacle does not render them as pages; this page is their documentation. `.common.env` holds the platform-independent settings (`TOOLS_DIR=tools`, the `fordaj/*` upstream repos, and the pinned `*_VERSION_DIR`s); each platform file sets `PROJECT_PLATFORM` and the `*_BIN` paths for its `<os>-<arch>`.

## Task scripts

- [version.sh](version.md) — print the repository version computed by versioner (`--format text|shell|json`).
- [docs.sh](docs.md) — generate the documentation site you are reading, with the bundled barnacle.
- [../release.md](../release.md) — version, fetch formatters, cross-compile, tag, and publish the GitHub release.
- [../demo.sh](../demo.md) — end-to-end tour of the CLI in a sandbox (never touches your tree).

## Related pages

- [tools/ — pinned helper binaries](../tools/README.md)
- [README.md](../README.md)
- [AGENTS.md](../AGENTS.md)

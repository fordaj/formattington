# Repository script tooling (`scripts/`)

```mermaid
flowchart TD
    Scripts["scripts/<br/>bash tooling"] --> Entry["entry.sh<br/>env cascade + log()"]
    Entry --> Env["env.sh<br/>.common.env + platform env"]
    Entry --> Version["version.sh<br/>versioner"]
    Entry --> Docs["docs.sh<br/>the documentation site"]
    Entry --> Release["release.sh<br/>dist + GitHub release"]
    Entry --> Demo["demo.sh<br/>CLI tour in a sandbox"]
    click Entry href "entry.sh.md"
    click Env href "env.sh.md"
    click Version href "version.sh.md"
    click Docs href "docs.sh.md"
    click Release href "release.sh.md"
    click Demo href "demo.sh.md"
```

The `scripts/` directory carries the repository's build/release/docs tooling: an environment cascade the other scripts source, plus one entry point per task. Everything here is bash; nothing requires a global install beyond what each script documents.

Vernacular used in the headers (`log INFO`, `$REPO_ROOT_DIR`, `VERSIONER_BIN`): read [entry.sh.md](entry.sh.md) first, then [env.sh.md](env.sh.md).

## The cascade (why the dotfiles exist)

- `scripts/entry.sh` — sources `scripts/env.sh` and defines `log()` (logamuffin, with a printf fallback). Sourced by everything else. See [entry.sh.md](entry.sh.md).
- `scripts/env.sh` — locates and sources the cascade below, then anchors the repository-relative `*_BIN` tool paths to absolute paths. See [env.sh.md](env.sh.md).
- `scripts/.common.env`, `scripts/.<os>-<arch>.env` — the cascade proper. Hidden files, so barnacle does not render them as pages; this page is their documentation. `.common.env` holds the platform-independent settings (`TOOLS_DIR=tools`, the `fordaj/*` upstream repos, and the pinned `*_VERSION_DIR`s); each platform file sets `PROJECT_PLATFORM` and the `*_BIN` paths for its `<os>-<arch>`.

## Task scripts

- [version.sh](version.sh.md) — print the repository version computed by versioner (`--format text|shell|json`).
- [docs.sh](docs.sh.md) — generate the documentation site you are reading, with the bundled barnacle.
- [release.sh](release.sh.md) — version, fetch formatters, cross-compile, tag, and publish the GitHub release.
- [demo.sh](demo.sh.md) — end-to-end tour of the CLI in a sandbox (never touches your tree).

## Related pages

- `tools/` — pinned helper binaries
- [README.md](../README.md)
- [AGENTS.md](../AGENTS.md)

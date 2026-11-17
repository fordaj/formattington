# The docs generator (`scripts/docs.sh`)

```mermaid
flowchart LR
    Repo("<img src='barnacle:file-text.svg' width='20' alt=''/><br/>repository .md / .mmd files") --> Barnacle("<img src='barnacle:package.svg' width='20' alt=''/><br/>BARNACLE_BIN<br/>pinned bundle under tools/barnacle")
    Cascade("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>env cascade<br/>scripts/.common.env</u>") --> Barnacle
    Barnacle --> Site("<img src='barnacle:folder.svg' width='20' alt=''/><br/>./site<br/>static HTML · git-ignored")
    class Cascade shFile
    class Barnacle boxNode
    class Site folder
    classDef shFile fill:#abe07a,stroke:#3f6212,color:#16280a
    classDef boxNode fill:#e8cf9d,stroke:#a16207,color:#33240a
    classDef folder fill:#f0c558,stroke:#8a6d1c,color:#332a05
    click Cascade href "env.sh.md"
```

Generates the documentation site you are reading: it runs the pinned `tools/barnacle/update.sh.md` binary against this repository.

Barnacle scans the working tree for Markdown (`.md`) and Mermaid (`.mmd`) files, renders each as a page, and keeps relative links between them. Generated output is static HTML; it is written to `./site/` by default and is never committed (it is in `.gitignore`).

## Usage

```bash
scripts/docs.sh                       # generate into ./site/
scripts/docs.sh --site ./public       # forward --site to barnacle
scripts/docs.sh --config ./docs/barnacle.json
scripts/docs.sh --help                # usage for this script
```

Any arguments other than `--help`/`-h` are forwarded to barnacle verbatim; see `barnacle --help` for `--config`/`--site` semantics. Barnacle exits non-zero and leaves no half-built site on failure; regenerating into a directory previously produced by barnacle wipes and rebuilds it.

## How it locates barnacle

Sources [entry.sh.md](entry.sh.md); `BARNACLE_BIN` comes from the [cascade](env.sh.md) (`tools/barnacle/v<version>/<platform>/barnacle[.exe]`). If the bundle is missing or not executable, the script fails with instructions to run `tools/barnacle/update.sh` (or set `BARNACLE_BIN` explicitly). It then runs barnacle with the repository root as the working directory, so barnacle's auto-discovered `./.barnacle.json` (optional) and its default `./site/` output land at the repository root regardless of the caller's cwd.

## What the site contains

Every `.md`/`.mmd` in the tree, including this one — see the [root README](../README.md) and [AGENTS.md](../AGENTS.md). `.git`, `node_modules`, `vendor`, `dist`, and the output directory are always excluded by barnacle; hidden files and directories (the cascade's `.common.env` et al.) are skipped unless `.barnacle.json` sets `includeHidden`.

## Related pages

- `../tools/barnacle/update.sh.md` — the pinned binaries and how to refresh them
- [entry.sh.md](entry.sh.md) · [env.sh.md](env.sh.md)

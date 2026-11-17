# Version resolution (`version.go`)

```mermaid
flowchart TD
    Cmd("<img src='barnacle:play.svg' width='20' alt=''/><br/>formattington --version") --> Memo("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>Version()<br/>memoized")
    Memo --> R{{"<img src='barnacle:hexagon.svg' width='20' alt=''/><br/>resolveVersion()<br/>highest wins"}}
    R -->|"1"| Env("<img src='barnacle:variable.svg' width='20' alt=''/><br/>FORMATTINGTON_VERSION<br/>explicit override")
    R -->|"2"| Stamp("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>buildVersion<br/>ldflags -X stamp at release time")
    R -->|"3"| Pkg("<img src='barnacle:file-text.svg' width='20' alt=''/><br/>PKG-INFO in cwd / repo root<br/>sdist-style builds")
    R -->|"4"| Versioner("<img src='barnacle:tag.svg' width='20' alt=''/><br/>bundled versioner, or versioner on PATH<br/>--repo-dir root --quiet --format json")
    Env --> Out("<img src='barnacle:type.svg' width='20' alt=''/><br/>the reported version string")
    Stamp --> Out
    Pkg --> Out
    Versioner --> Out
    class Memo goFile
    class R hexNode
    class Stamp goFile
    class Versioner tagNode
    classDef goFile fill:#8fd8ef,stroke:#0277a8,color:#08303d
    classDef hexNode fill:#ffe08a,stroke:#a16207,color:#422006
    classDef tagNode fill:#f6b8d4,stroke:#be185d,color:#500724
```

`version.go` makes `formattington --version` report the same version the release was tagged with.

## Resolution order

`Version()` (line ~41) memoizes `resolveVersion()` (line ~48), which prefers, highest first:

1. `FORMATTINGTON_VERSION`, if set — an explicit override for CI and testing.
2. `buildVersion` (line ~24) — the version stamped into release binaries at link time (`-ldflags "-X ...internal/formattington.buildVersion=<version>"`), so a binary distributed outside the repository still reports its release tag. Empty for local builds.
3. `versionFromPkgInfo()` (line ~78) — the `Version:` line of a `PKG-INFO` file in the working directory or repository root (sdist-style builds).
4. `bundledVersioner` + `versionFromVersioner` (lines ~104, ~125) — the pinned `tools/versioner/update.sh.md` bundle under `tools/versioner/` (resolved via the shared `findBundledForHost`), or `versioner` on PATH, run as `versioner --repo-dir <root> --quiet --format json` against the repository root.
5. The fallback `0.0.0` with a warning on stderr when none of the above is available.

## Why not a constant

Release binaries must report the exact tag their archive was built from, even outside a repository. Computing the version with `tools/versioner/update.sh.md` at link time — and again at run time as a fallback — keeps one source of truth shared with `scripts/version.sh` and the release pipeline.

## Related pages

- [scripts/version.sh](../../scripts/version.sh.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Command dispatch and the tool registry](cli.go.md)

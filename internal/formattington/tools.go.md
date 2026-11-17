# Tool discovery and subprocess helpers (`tools.go`)

```mermaid
flowchart TD
    Need("<img src='barnacle:puzzle.svg' width='20' alt=''/><br/>integration needs an executable") --> Find("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>FindTool")
    Find --> Override("<img src='barnacle:variable.svg' width='20' alt=''/><br/>1 · FORMATTINGTON_FORMATTERS_DIR<br/>environment override")
    Find --> Bundle("<img src='barnacle:folder.svg' width='20' alt=''/><br/>2 · FormattersDir()<br/>checkout formatters/ or unpacked embedded cache<br/>newest numeric version wins")
    Find --> PathLookup("<img src='barnacle:search.svg' width='20' alt=''/><br/>3 · PATH via findOnPath<br/>name, then name.exe on Windows")
    Override -->|unset or missing| Bundle
    Bundle -->|no bundle matches| PathLookup
    PathLookup -->|not found| Missing("<img src='barnacle:circle-exclamation-point.svg' width='20' alt=''/><br/>ExitNotFound 127")
    Bundle --> Exe("<img src='barnacle:binary.svg' width='20' alt=''/><br/>resolved executable")
    PathLookup --> Exe
    Exe --> Run("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>runTool()<br/>arguments as a slice · no shell · captures exit code</u>")
    Bundle -.-> Embedded("<img src='barnacle:package.svg' width='20' alt=''/><br/><u>full-release payload<br/>unpacked on first use</u>")
    class Find goFile
    class Bundle folder
    class Missing errNode
    class Run goFile
    class Embedded boxNode
    classDef goFile fill:#8fd8ef,stroke:#0277a8,color:#08303d
    classDef folder fill:#f0c558,stroke:#8a6d1c,color:#332a05
    classDef errNode fill:#f6b8b8,stroke:#b91c1c,color:#450a0a
    classDef boxNode fill:#e8cf9d,stroke:#a16207,color:#33240a
    click Embedded href "embedded.go.md"
    click Run href "cli.go.md"
```

`tools.go` resolves executables for every integration, and owns the shared subprocess plumbing.

## Executable resolution

`FindTool(name, version, verbose)` (line ~286) resolves an integration's executable:

1. `FORMATTINGTON_FORMATTERS_DIR` environment override, if set.
2. The bundled formatter tree — the directory returned by `FormattersDir()` (line ~116), which is either a checkout's `formatters/` (executable-adjacent or ancestor) or the per-user cache into which full-release embedded payloads are unpacked (see [embedded.go.md](embedded.go.md)).
3. `PATH` via `findOnPath` (line ~268); on Windows, both `name` and `name.exe` are tried.

Bundled versions are compared numerically with `versionKey` (line ~178) — `v10` beats `v9` and `v23.1.3` beats `v22.1.8`. `bundledVersions` (line ~221) lists candidate version directories; `findBundledForHost` (line ~242) prefers an explicitly requested version or the newest one carrying a host-platform executable and returns both the path and the version (shared with `list` and the versioner fallback). `hostExecutableName` maps a tool name onto its host executable name (`.exe` on Windows). A missing tool is reported as `ExitNotFound` (`127`); `--verbose` (or `FORMATTINGTON_VERBOSE`) prints the resolved path.

`PlatformDir()` (line ~95) maps the host to the hyphenated `<os>-<arch>` bundle directory name (`linux-x86_64`, `macos-arm64`, `windows-x86_64`, …) used everywhere under `formatters/` and `tools/`.

## Subprocess helpers

`runTool(executable, arguments, captureOutput)` (line ~332) runs the tool with arguments passed as a slice (never a shell string), wiring stdin/stdout/stderr and returning the raw exit code. `captureOutput` asks for output buffering (used by check modes that derive status from stdout, like gofmt's `-l`); otherwise output streams through to the caller. `IsInformational` (line ~59) recognizes help/version-style invocations so discovery does not sprinkle file lists onto them.

Also here: `iterChunks` (batching respecting command-line limits), `contains`, `pathSuffix`, and `repoRoot`/`findRootWithBundles` — the ancestor-directory search for a `formatters/` tree when the binary itself is not inside a checkout.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [File discovery](discovery.go.md)
- [Embedded formatter payload unpacking (full releases)](embedded.go.md)

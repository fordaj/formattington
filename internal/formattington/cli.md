# Command dispatch and the tool registry (`cli.go`)

`cli.go` is the canonical command-line entry point (`Main`) and the sole registry of formatter integrations. Every other file in the package either supports dispatch (discovery, tool resolution, configuration) or implements one integration.

## The registry

`registry` (line ~21) maps each tool name to its `Formatter` implementation. Its order is deliberate — it is the order `formattington all` runs in and the order exit codes are aggregated in. The order is the Stack Overflow Developer Survey 2026 popularity ranking (see [AGENTS.md — presentation order](../../AGENTS.md)), not alphabetical.

`languageAliases` (line ~68) maps language and file-type names onto registered tools — for example `md`/`markdown`/`html` → `prettier`, `python`/`py` → `black`, `c`/`cpp`/`c++` → `clang-format`. Unrecognized words are a usage error (exit 2); an alias whose backend is not registered reports `'<name>' is not yet supported`.

The registry order is mirrored by [init.json](init.md), the hardcoded order lists in [help.md](help.md), [README.md](../../README.md), and [AGENTS.md](../../AGENTS.md) — change them together.

## Dispatch flow

`Main(argv)` (line ~137):

1. Parses global options with `parseGlobalArgs` — `--verbose`, `--check`, `--dry-run`, `--config`, `--series`. Selecting a tool must come from the first bare token that is not consumed as an option value (`findFormatterIndex`, line ~326).
2. Dispatches to `list`/`init`, to `all` (registry loop, concurrent with `--series` for sequential), to a registered tool by name, or to a language alias.
3. Runs each tool through `runFormatter` (line ~447): resolves embedding/discovery (`DiscoverFiles`), batches by `maxFilesPerInvocation`, calls the integration with `Options`.
4. Aggregates results with `aggregateCodes` (line ~511) — the last non-zero code wins; `0` only if all succeed. Missing tools become exit `127` but never prevent other formatters from running.

`runFormatter` discovers files itself when the user supplied no explicit file arguments: matching extensions for that formatter are appended to the invocation in batches. Explicit file arguments are passed verbatim (`hasUserFileArgs`, line ~488). `IsInformational` check/version passthroughs (`--help`, `--version`) are never given discovered files.

Results for `all` on default invocations run concurrently (`runParallel`, registration order for aggregation, interleaved output possible); `--series` runs sequentially in registration order (`runSeries`). Configuration-file commands run through `runConfigParallel`/`runConfigSeries` (line ~591).

## Options carried to integrations

`Options` (see [formatter.md](formatter.md)) is the per-run bundle: `Check`, `DryRun`, `Args` (forwarded verbatim), `Verbose`.

## Related pages

- [Tool discovery, bundling, and shared subprocess helpers](tools.md)
- [File discovery](discovery.md)
- [Configuration file loading (`.formattington.json`)](config.md)
- [Execution options and the Formatter contract](formatter.md)
- [Architecture diagram](architecture.mmd)
- [list command](list.md) · [init command](init.md) · [help output](help.md) · [version resolution](version.md)

# The end-to-end tour (`demo.sh`)

`demo.sh` runs a scripted, narrated tour of Formattington against throwaway files — a safe way to see the CLI's behavior end to end.

## What it does

- Creates a temporary sandbox (including a path with spaces) with deliberately messy sample files, runs every demonstrated `formattington` invocation against explicit paths inside the sandbox, and removes the sandbox on exit. **It never modifies the repository.**
- Every formattington call is non-destructive: `--help`, `--version`, `--verbose`, `--check`, or `--dry-run`. Discovery is shown by running from the sandbox so `all` stays scoped to the demo files.
- Narration goes through `log <level> <message>` from [scripts/entry.sh](scripts/entry.md) (logamuffin, with printf fallback); `LOG_LEVEL=TRACE` shows the per-call detail. The `LOG_SOURCE` field shows this script's file:line as messages log.

## Usage

```bash
./demo.sh          # run the whole tour (single non-interactive pass)
./demo.sh --help   # usage
```

## Prerequisites and environment

- `FORMATTINGTON_BIN` — path to the formattington binary; defaults to `./bin/formattington`, then PATH. Build it with `make build`.
- `VERSIONER_BIN`, `LOGAMUFFIN_BIN` — resolution via [scripts/env.sh](scripts/env.md), then PATH.
- `LOG_LEVEL` — minimum log level (default INFO).
- `FORMATTINGTON_VERBOSE=1` — show resolved executables.
- PATH-only tools (`black`, `prettier`, `pwsh`) may exit 127 when missing; the tour detects that and continues.

## Related pages

- [scripts/README.md](scripts/README.md)
- [README.md](README.md)

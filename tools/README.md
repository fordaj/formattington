# Bundled formatter binaries

Formattington commits pinned formatter binaries here intentionally
(policy (a) in #11). `find_tool()` in `src/formattington/_tools.py`
looks here first, then falls back to `PATH`.

## Layout

```text
tools/
  <name>/
    v<version>/
      <platform>/
        <executable>
```

- `<name>` is `shfmt`, `clang-format`, `taplo`, or `versioner`.
- `<version>` is the upstream version without the leading `v`
  in code (for example `v3.14.1`).
- `<platform>` is the hyphenated `<os>-<arch>` from
  `get_platform_dir()` (for example `macos-arm64`,
  `linux-x86_64`, `windows-x86_64`).
- `<executable>` is `shfmt` / `clang-format` / `taplo` / `versioner`, with `.exe` on
  Windows.

## Pinned versions

| Tool | Pinned version | Upstream | Platforms in this repo |
| --- | --- | --- | --- |
| shfmt | v3.14.1 | https://github.com/mvdan/sh | linux-arm64, linux-x86_64, macos-arm64, macos-x86_64, windows-arm64, windows-x86_64 |
| clang-format | v23.1.3 | LLVM project (clang-format) | linux-arm64, linux-x86_64, macos-arm64, windows-arm64, windows-x86_64 |
| clang-format (legacy) | v22.1.8 | LLVM project (clang-format) | macos-x86_64 only (superseded by v23.1.3) |
| taplo | v0.10.0 | https://github.com/tamasfe/taplo (taplo-cli) | linux-arm64, linux-x86_64, macos-arm64, macos-x86_64, windows-arm64, windows-x86_64 |
| versioner | v0.0.3.0.5 | https://github.com/fordaj/Versioner | linux-arm64, linux-x86_64, macos-arm64, macos-x86_64, windows-x86_64 |

Versions are compared numerically, so `v23.1.3` wins over `v22.1.8`
and `v10` would win over `v9`. The legacy v22.1.8 bundle is retained
only for reference; the resolver prefers the newest version.

## Refresh

1. Download the new upstream release for each platform from the
   upstream table above.
2. Extract into the `tools/<name>/v<version>/<platform>/` layout
   (use `tools/clang-format/unzip.sh` for `*.tar.xz` clang-format
   archives; `tools/clang-format/clean.sh` shows which files are
   kept from an extraction). Taplo releases ship one gzipped binary
   per platform (`taplo-<os>-<arch>.gz`); gunzip and place it as
   `taplo` (or `taplo.exe` on Windows) under
   `tools/taplo/v<version>/<platform>/`. Versioner refreshes are
   automated with `tools/versioner/update.bash`, which downloads the
   latest release with `gh`, verifies `checksums.txt`, extracts each
   binary, and removes old `v*/` dirs.
3. Keep the exact version in the directory name (`v<version>`) and
   update the table above plus the note in the top-level `README.md`
   in the same commit.
4. Verify resolution:
   ```bash
   formattington --verbose shfmt
   formattington --verbose clang-format
   formattington --verbose taplo
   # or: FORMATTINGTON_VERBOSE=1 formattington taplo
   ```
   The verbose output prints the resolved path and whether it came
   from the bundle or `PATH`.

`versioner` is not a formatter; `tools/versioner/update.bash` verifies
the host binary (`--version` contains the tag) on each run.

There is no on-demand fetch. Refreshes are manual commits so binary
changes are reviewable.

## Size and fallback

`tools/` is about 120 MB. Committing binaries makes a fresh clone
usable offline, at the cost of clone/download size. If a platform
bundle is absent, `find_tool()` falls back to `PATH` without any
code change; if both fail it raises `FileNotFoundError`, which the
CLI prints as `formattington: Could not find ...` and exits `127`.

## Hygiene

Only binaries and the helper scripts (`clean.sh`, `unzip.sh`,
`update.bash`) are tracked. `__pycache__/`, `*.pyc`, and `.DS_Store` are ignored via
the top-level `.gitignore` and must not be tracked.

# Embedded formatter payloads (`embedded.go`, `embedded_full.go`, `embedded_lite.go`)

The release ships two binaries per platform. The **lite** binary is just Formattington; the **full** binary embeds its platform's formatter executables and unpacks them on first use.

## How the payload travels

`embedded_full.go` (build tag `full`) carries `assets/formatters.zip` — one archive holding the whole `<tool>/v<version>/<platform>/<executable>` tree for the build platform. `assets/` is not committed; `scripts/release.sh` fills it before compiling with `-tags full`. `embedded_lite.go` (build tag `!full`) declares the same variable empty.

`go:embed` cannot execute embedded bytes in place, so `embedded.go` unpacks the archive into a per-user cache directory on first use:

`installEmbeddedFormatters` (line ~93) extracts the payload under `embeddedFormattersCacheDir()` (line ~64) — `formatters/` under `os.UserCacheDir()` (for example `~/Library/Caches/formattington/formatters` on macOS) — keyed by a `zipFingerprint` (line ~255) of the embedded archive's contents. `extractedFormattersMatch` (line ~266) reuses an existing extraction when fingerprints agree and replaces it wholesale otherwise, so upgrading the full binary never serves stale tools. `collectGarbage` (line ~175) prunes superseded cache entries; `secureMkdirAll` (line ~76) and `regularZipFiles`/`extractZipMember` guard the extraction against pathological zip entries.

## Resolution order

`FormattersDir()` in [tools.md](tools.md) prefers a `FORMATTINGTON_FORMATTERS_DIR` override, then the unpacked embedded payload (full builds), then a `formatters/` directory in a checkout (executable-adjacent or ancestor), then `PATH`. Lite builds skip the embedded branch entirely.

## Running the payload out of tree

The full release binary smoke-tests its embedded formatters before publishing: with a stripped `PATH` and an empty `HOME`, it invokes each embedded formatter with `--version`. If the embedded payload cannot unpack or run on the release host, the release fails.

## Related pages

- [Tool discovery, bundling, and shared subprocess helpers](tools.md)
- [Command dispatch and the tool registry](cli.md)

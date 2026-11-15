# scalafmt (`scalafmt`)

scalafmt is the standard Scala formatter, covering `.scala`, `.sbt`, and `.sc` (worksheet) files. It is PATH-only because the upstream native binaries cannot honor a version-pinned configuration.

## At a glance

| | |
| --- | --- |
| Executable | `scalafmt` |
| Resolution | PATH-only (via coursier, `cs install scalafmt`, or `brew install scalafmt`; JVM launcher) |
| Extensions | `.scala`, `.sbt`, `.sc` |
| Operations | Format, Check |
| Language aliases | `scala` -> `scalafmt` |
| Native configuration | `.scalafmt.conf` (requires a `version = <x.y.z>` line since 3.1.0) |

## Behavior

- Default invocation: `scalafmt` — format in place.
- `--check`: `scalafmt --test` — print a diff and exit 1 when formatting is needed (0 when clean, no writes).
- `--dry-run`: same as `--check` (`--test` is scalafmt's diff mode).
- Forwarded arguments: verbatim; `--config` is honored, and a config is never synthesized by Formattington.
- Exit codes: no remapping. Exit codes are a bitfield (Ok=0, TestError=1, ParseError=2, CmdLineArg=4, Unexpected=8, UnsupportedVersion=16, OR-ed); all bits pass through verbatim.

## Notes

- scalafmt requires a `version = <x.y.z>` line in `.scalafmt.conf`. The upstream native release binaries have no dynamic runner, so a config whose pinned version differs from the compiled-in one fails hard (`error: invalid Scalafmt version.`, exit bit 16) — hence no bundle is shipped.
- The coursier/JVM launcher honors the pin by downloading it dynamically — the only route with full fidelity — which is why PATH-only is the only supported resolution.
- The `.scalafmt.conf` `version =` line makes the coursier/JVM launcher download that scalafmt version at run time (remote fetch) — see the untrusted-repositories guidance in [README.md](../../README.md).
- The `.sc` worksheet extension is covered.

## Related pages

- [Command dispatch and the tool registry](cli.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.md)
- [Execution options and the Formatter contract](formatter.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/scalameta/scalafmt

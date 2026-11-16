# Execution options and the Formatter contract (`formatter.go`)

```mermaid
flowchart LR
    Opts["Options<br/>Check · DryRun · Args · Verbose"] --> Contract["Formatter<br/>func(Options) (int, error)"]
    Contract --> Impls["one integration per tool<br/>registered in cli.go"]
    Contract --> Codes["returned exit code<br/>0 ok · 1 ExitFormatRequired<br/>127 missing tool · 2 usage error"]
    Impls -->|forwarded verbatim| Args[Args<br/>tool-native flags]
    click Impls href "cli.go.md"
    click Contract href "cli.go.md"
```

`formatter.go` holds the two types every integration shares plus the shared integration opening.

## `Options`

| Field     | Meaning                                                                                                                                                                                                                                      |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Check`   | Report violations without modifying files.                                                                                                                                                                                                   |
| `DryRun`  | Validate without modifying files (most tools alias this to their check mode; a few have no native check mode and synthesize one — see the integration docs, e.g. [mojo_format.go.md](mojo_format.go.md) and [odinfmt.go.md](odinfmt.go.md)). |
| `Args`    | Arguments forwarded verbatim to the underlying tool.                                                                                                                                                                                         |
| `Verbose` | Report the resolved executable on stderr.                                                                                                                                                                                                    |

## `Formatter`

```go
type Formatter func(Options) (int, error)
```

The returned integer is the process exit code the CLI reports. A non-nil error means the executable could not be resolved or started — the CLI turns that into exit `127`. Integrations that want "files need formatting" reported as exit `1` use the shared `ExitFormatRequired` value (defined in the CLI; see [cli.go.md](cli.go.md)); anything else passes through unchanged so meaningful upstream diagnostics survive.

`withArgs(prefix, args)` prefixes a tool's mandatory flags (for example gofmt's mode flag) in front of forwarded user arguments, always on a copy.

`beginIntegration(toolName, opts, prefix)` is the shared opening of every integration: it resolves the executable (`FindTool`), and — when the forwarded arguments are an informational help/version passthrough (`IsInformational`, with `prefix` applied, for example `dart format --help`) — runs the tool immediately and reports `done`. Callers return `(code, err)` as-is, which yields exit `127` for a missing tool and the tool's own exit code for the passthrough; otherwise they receive the executable and a fresh copy of the arguments (without `prefix`) and layer their mode flags on. A non-nil error means the executable could not be resolved.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [AGENTS.md](../../AGENTS.md)

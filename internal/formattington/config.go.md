# Configuration files (`config.go`)

```mermaid
flowchart TD
    Resolve("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>resolveConfigPath") --> Precedence{{"<img src='barnacle:hexagon.svg' width='20' alt=''/><br/>precedence"}}
    Precedence -->|"1"| Explicit("<img src='barnacle:braces.svg' width='20' alt=''/><br/>--config PATH")
    Precedence -->|"2"| EnvVar("<img src='barnacle:variable.svg' width='20' alt=''/><br/>FORMATTINGTON_CONFIG_PATH")
    Precedence -->|"3"| Cwd("<img src='barnacle:braces.svg' width='20' alt=''/><br/>./.formattington.json<br/>cwd only · no parent search")
    Explicit --> Load("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>loadConfig")
    EnvVar --> Load
    Cwd --> Load
    Load --> Validate("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>validateConfig<br/>commands: array of string arrays<br/>first element a registered tool or alias")
    Validate -->|"invalid / missing file"| Exit2("<img src='barnacle:circle-exclamation-point.svg' width='20' alt=''/><br/><u>exit 2 — usage error · never silent</u>")
    Validate -->|"valid"| Run("<img src='barnacle:play.svg' width='20' alt=''/><br/><u>commands run with<br/>discovery as usual</u>")
    class Resolve goFile
    class Precedence hexNode
    class Explicit jsonFile
    class Cwd jsonFile
    class Load goFile
    class Validate goFile
    class Exit2 errNode
    classDef goFile fill:#8fd8ef,stroke:#0277a8,color:#08303d
    classDef hexNode fill:#ffe08a,stroke:#a16207,color:#422006
    classDef jsonFile fill:#cdb9f7,stroke:#6d28d9,color:#2e1065
    classDef errNode fill:#f6b8b8,stroke:#b91c1c,color:#450a0a
    click Run href "cli.go.md"
    click Exit2 href "cli.go.md"
```

`config.go` loads and validates the optional `.formattington.json` that rebinds which commands `formattington` runs for each language — for example wiring `formattington all` to a project's own set of invocations.

## What it accepts

A configuration file carries a `commands` array of argument arrays (this repository's own file is a representative example). Each entry names a registered tool followed by the arguments to pass it. The default invocation (discovery supplies matching files, as usual) is what `init` writes; explicit `[...]` argument lists are appended verbatim when present.

Validation (`validateConfig`) checks the shape — a JSON object with only a `commands` key, an array of string arrays whose first element is a registered tool name or alias; anything else is a located error (`pyRepr` renders values in Python `repr()` style, so validation messages stay familiar to Python-literate readers). A missing `commands` key and unknown keys are both rejected, as are unknown tools and aliases that resolve to unregistered backends (`'<alias>' is not yet supported`). `commands: []` is valid and runs nothing. String (shell-style) entries and empty command lists are rejected — configuration is never passed through a shell.

`resolveConfigPath` (line ~27) picks the path by precedence: explicit `--config PATH` wins, then `FORMATTINGTON_CONFIG_PATH`, then `./.formattington.json` in the working directory only (no parent search). `loadConfig` (line ~171) returns the commands, the source label (`--config`, `FORMATTINGTON_CONFIG_PATH`, or `auto-discovered .formattington.json`), and the path for error messages. Missing or malformed configuration is a usage error (exit `2`), never a silent fallback; no config means the full registry-driven `all` behaviour.

## Related pages

- [init command](init.go.md) — writes this file; the embedded catalogue it draws from is [init.json](init.go.md)
- [Command dispatch and the tool registry](cli.go.md)
- [README.md — configuration section](../../README.md)

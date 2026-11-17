# gdformat (`gdformat`)

```mermaid
flowchart LR
    Src("<img src='barnacle:file-code.svg' width='20' alt=''/><br/>.gd<br/>covered files") --> Reg("<img src='barnacle:file-code.svg' width='20' alt=''/><br/><u>formattington gdformat<br/>cli.go registry entry</u>")
    Reg --> Tool("<img src='barnacle:terminal.svg' width='20' alt=''/><br/><u>gdformat<br/>PATH only</u>")
    Tool --> Ops("<img src='barnacle:list-checks.svg' width='20' alt=''/><br/><u>Operations: Format, Check</u>")
    click Reg href "cli.go.md"
    click Tool href "tools.go.md"
    click Ops href "formatter.go.md"
    class Src goFile
    class Reg goFile
    class Ops checkNode
    classDef goFile fill:#8fd8ef,stroke:#0277a8,color:#08303d
    classDef checkNode fill:#9fe8c9,stroke:#0f766e,color:#052e22
```

gdformat (github.com/Scony/godot-gdscript-toolkit) is the Godot 4 GDScript formatter, from the gdtoolkit Python package. Formattington registers it for `.gd` files.

## At a glance

|                      |                                                                                                 |
| -------------------- | ----------------------------------------------------------------------------------------------- |
| Executable           | `gdformat`                                                                                      |
| Resolution           | PATH-only (pure-Python package with no release binaries; on all six targets)                    |
| Extensions           | `.gd`                                                                                           |
| Operations           | Format, Check                                                                                   |
| Language aliases     | `gdscript` → `gdformat`                                                                         |
| Native configuration | `gdformatrc` / `.gdformatrc`, discovered by walking up from the working directory (tool-native) |

## Behavior

- Default invocation: `gdformat` — format in place
- `--check`: `gdformat -c` — print "would reformat <file>" and exit 1 when changes are needed, 0 otherwise
- `--dry-run`: `gdformat -d` — print a unified diff, do not modify (implies `--check`)
- Forwarded arguments: forwarded verbatim to `gdformat`
- Exit codes: no remapping — `-c`/`-d` exit 1 when reformatting is needed, as reported by the tool

## Notes

- Install with `pipx install gdtoolkit==4.*` for the Godot 4 line; gdtoolkit 3.x targets Godot 3. It is PATH-only because there are no release binaries.
- Formattington deliberately does not set `--fast`, so the safety checks (prologue/epilogue validation) stay on.
- `-d`/`--diff` prints a unified diff and implies `--check`.
- Configuration (`gdformatrc` / `.gdformatrc`) is discovered by walking up from the working directory and is tool-native.

## Related pages

- [Command dispatch and the tool registry](cli.go.md)
- [Tool discovery, bundling, and shared subprocess helpers](tools.go.md)
- [Execution options and the Formatter contract](formatter.go.md)
- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- Upstream: https://github.com/Scony/godot-gdscript-toolkit

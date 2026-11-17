# Process entry point (`cmd/formattington/main.go`)

```mermaid
flowchart LR
    OS("<img src='barnacle:terminal.svg' width='20' alt=''/><br/>os.Args[1:]") --> Main("<img src='barnacle:play.svg' width='20' alt=''/><br/>formattington.Main()")
    Main --> Code("<img src='barnacle:binary.svg' width='20' alt=''/><br/>process exit code")
    Main --> Impl("<img src='barnacle:folder.svg' width='20' alt=''/><br/><u>all logic lives in<br/>internal/formattington</u>")
    click Impl href "../../internal/formattington/README.md"
    class Impl folder
    classDef folder fill:#f0c558,stroke:#8a6d1c,color:#332a05
```

`main.go` is the process boundary: it calls `formattington.Main(os.Args[1:])` and exits with the returned code. There is no logic worth testing here — everything testable lives in `internal/formattington/` and is covered by `go test ./internal/formattington`.

```go
func main() {
	os.Exit(formattington.Main(os.Args[1:]))
}
```

The split exists so the whole CLI can be driven in-process from tests without a re-exec.

## Where to go next

- [Command dispatch and the tool registry](../../internal/formattington/cli.go.md)
- [Architecture diagram](../../internal/formattington/architecture.mmd)

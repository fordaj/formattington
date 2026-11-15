# The `internal/formattington` package

All of Formattington's implementation lives in this package: the CLI, the tool registry, discovery, configuration, and every formatter integration. `cmd/formattington/main.go` is a two-line shim over `Main` here.

If you want the big picture first, open the [architecture diagram](architecture.mmd), then [cli.md](cli.md). To refresh the bundled helper binaries, see [tools/README.md](../../tools/README.md); to render this documentation, see [scripts/docs.md](../../scripts/docs.md).

## The contract with the integrations

Every integration is a function with the signature `func(Options) (int, error)` ([formatter.md](formatter.md)), registered by name in `cli.go`'s `registry` ([cli.md](cli.md)), resolving its executable through `FindTool` ([tools.md](tools.md)), and receiving pre-discovered files from `DiscoverFiles` ([discovery.md](discovery.md)). Exit codes above `0` are upstream exit codes; `1` is reserved for "files need formatting" (`ExitFormatRequired`), `127` for a missing tool, and `2` for usage errors.

## Core pages

- [Command dispatch and the tool registry — cli.go](cli.md)
- [Tool discovery and subprocess helpers — tools.go](tools.md)
- [Execution options and the Formatter contract — formatter.go](formatter.md)
- [File discovery — discovery.go](discovery.md)
- [Configuration files — config.go](config.md)
- [Help output — help.go](help.md)
- [Version resolution — version.go](version.md)
- [The `list` command — list.go](list.md)
- [The `init` command and embedded catalogue — init.go](init.md)
- [Embedded formatter payloads — embedded.go](embedded.md)
- [Architecture diagram — architecture.mmd](architecture.mmd)
- [Process entry point — cmd/formattington/main.go](../../cmd/formattington/main.md)

## Integration pages (registration order)

The order mirrors the registry — Stack Overflow Developer Survey 2026 popularity order (see [AGENTS.md](../../AGENTS.md)), not alphabetical:

| Tool               | Language(s)                       | Docs                                           |
| ------------------ | --------------------------------- | ---------------------------------------------- |
| prettier           | JS/TS/CSS/HTML/Markdown/JSON/YAML | [prettier.md](prettier.md)                     |
| sqlfluff           | SQL                               | [sqlfluff.md](sqlfluff.md)                     |
| black              | Python                            | [black.md](black.md)                           |
| shfmt              | Shell                             | [shfmt.md](shfmt.md)                           |
| google-java-format | Java                              | [google_java_format.md](google_java_format.md) |
| dotnet-format      | C#/VB.NET                         | [dotnet_format.md](dotnet_format.md)           |
| powershell         | PowerShell                        | [powershell.md](powershell.md)                 |
| clang-format       | C/C++/Obj-C/…                     | [clang_format.md](clang_format.md)             |
| rustfmt            | Rust                              | [rustfmt.md](rustfmt.md)                       |
| php-cs-fixer       | PHP                               | [php_cs_fixer.md](php_cs_fixer.md)             |
| gofmt              | Go                                | [gofmt.md](gofmt.md)                           |
| goimports          | Go                                | [goimports.md](goimports.md)                   |
| ktfmt              | Kotlin                            | [ktfmt.md](ktfmt.md)                           |
| stylua             | Lua                               | [stylua.md](stylua.md)                         |
| swift-format       | Swift                             | [swift_format.md](swift_format.md)             |
| rubocop            | Ruby                              | [rubocop.md](rubocop.md)                       |
| dart-format        | Dart                              | [dart_format.md](dart_format.md)               |
| styler             | R                                 | [styler.md](styler.md)                         |
| nixfmt             | Nix                               | [nixfmt.md](nixfmt.md)                         |
| gdformat           | GDScript                          | [gdformat.md](gdformat.md)                     |
| npm-groovy-lint    | Groovy                            | [npm_groovy_lint.md](npm_groovy_lint.md)       |
| perltidy           | Perl                              | [perltidy.md](perltidy.md)                     |
| zig-fmt            | Zig                               | [zig_fmt.md](zig_fmt.md)                       |
| mix-format         | Elixir                            | [mix_format.md](mix_format.md)                 |
| fourmolu           | Haskell                           | [fourmolu.md](fourmolu.md)                     |
| scalafmt           | Scala                             | [scalafmt.md](scalafmt.md)                     |
| fprettify          | Fortran                           | [fprettify.md](fprettify.md)                   |
| erlfmt             | Erlang                            | [erlfmt.md](erlfmt.md)                         |
| jlfmt              | Julia                             | [jlfmt.md](jlfmt.md)                           |
| cljfmt             | Clojure                           | [cljfmt.md](cljfmt.md)                         |
| gleam-format       | Gleam                             | [gleam_format.md](gleam_format.md)             |
| gnatformat         | Ada                               | [gnatformat.md](gnatformat.md)                 |
| fantomas           | F#                                | [fantomas.md](fantomas.md)                     |
| ocamlformat        | OCaml                             | [ocamlformat.md](ocamlformat.md)               |
| forge-fmt          | Solidity                          | [forge_fmt.md](forge_fmt.md)                   |
| mojo-format        | Mojo                              | [mojo_format.md](mojo_format.md)               |
| odinfmt            | Odin                              | [odinfmt.md](odinfmt.md)                       |
| elm-format         | Elm                               | [elm_format.md](elm_format.md)                 |
| markdownlint       | Markdown (lint)                   | [markdownlint.md](markdownlint.md)             |
| taplo              | TOML                              | [taplo.md](taplo.md)                           |

## Testing

Tests live next to the code as `*_test.go` and run without network access or installed formatter binaries (fake executables in temporary directories; tool discovery is pointed at stub trees). Run them with `make test` (or `make check` for vet + gofmt + tests). The suite also round-trips `init.json` against every registered tool.

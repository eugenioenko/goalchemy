# Using Goalchemy

Goalchemy compiles programs written in a restricted subset of Go into other languages. Source files are ordinary `.go` files: they build and run with the Go toolchain, and Goalchemy accepts the subset described in [the language specification](../specs/language.md). Release 0.1 supports the sequential and cooperative language on six targets: lowered Go, TypeScript for Node.js, Python 3.10+, Java 21, C# for .NET 8, and Rust (edition 2021). Run `goalchemy features` for the current feature matrix.

## Install

```sh
go build -o bin/goalchemy ./cmd/goalchemy
```

The repository's `go.mod` selects the Go 1.27.1 reference toolchain. Source programs must declare `go 1.25` or earlier. TypeScript output needs Node.js 22.6 or later, because it runs `.ts` files directly through Node's type stripping.

## Commands

| Command | Purpose |
| --- | --- |
| `goalchemy check [-gate g] [-tags t] [-json] [packages]` | Load, type-check, and validate against the language gate. |
| `goalchemy compile -target <go\|typescript\|python\|java\|csharp\|rust\|ir> -out <dir> [packages]` | Write a complete, runnable target directory. |
| `goalchemy build [-config goalchemy.yaml]` | Compile every target listed in a project configuration. |
| `goalchemy run -target <name> [packages]` | Compile to a temporary directory and run the program. |
| `goalchemy features` | Print supported features and targets. |
| `goalchemy spec validate` / `spec generate [-check]` / `test` | Maintain and verify the runtime contract catalog. |

## Project configuration

`goalchemy build` reads `goalchemy.yaml` from the current directory:

```yaml
schema_version: 1
packages: [./cmd/app]      # package patterns, relative to this file
tags: [prod]               # optional build tags
gate: sequential           # sequential (default) or cooperative
targets:
  go: {out: out/go}
  typescript: {out: out/ts}
  python: {out: out/py}
  java: {out: out/java}
  csharp: {out: out/cs}
  rust: {out: out/rs}
```

The file uses the same restricted YAML as the contract catalog: no anchors, aliases, or floats, and no unknown keys.

## Output

Each target directory contains the generated program, the runtime files it needs (one file per runtime function, plus shared representation files), a `README.md`, and `goalchemy.manifest.json`. The manifest records the source profile, compiler version, contract IDs, versions, and hashes, and every runtime and generated file. Output is deterministic for the same inputs.

- **Go**: `main.go`, `go.mod`, `rt/`. Run with `go run .`. Line directives map positions back to the Goalchemy source.
- **TypeScript**: `main.ts`, `main.ts.map`, `rt/`, `package.json`. Run with `node main.ts`, and add `--enable-source-maps` for source positions in stack traces.
- **Python**: `main.py`, `rt/` (a package), and `main.py.lines`, which maps generated lines to Goalchemy source positions. Run with `python3 main.py`; Python 3.10 or later is required.
- **Java**: `Main.java`, `rt/`, `run.sh`, and `Main.java.lines` (generated lines to source positions). Run with `sh run.sh`, which compiles with `javac` and runs on a Java 21 or later JDK (`JAVA_HOME` is honored).
- **C#**: `Main.cs`, `rt/`, `main.csproj`, `run.sh`, and `Main.cs.lines`. Run with `sh run.sh`, which compiles with the .NET 8 SDK's C# compiler and runs on .NET 8 (`DOTNET_ROOT` is honored); `dotnet run` also works.
- **Rust**: `src/main.rs`, `src/rt/`, `Cargo.toml`, `run.sh`, and `src/main.rs.lines`. Run with `sh run.sh` (plain `rustc`, standard library only) or `cargo run --release`. Values live in a traced heap collected at safepoints; set `GOALCHEMY_HEAP_STATS=1` to print heap statistics at exit and `GOALCHEMY_GC_THRESHOLD=<n>` to collect more often.

When the working directory or a parent has a `.toolchains` directory holding `jdk-*` or `dotnet`, `goalchemy run` and the test suites use those toolchains.

Programs write output with Go's `print` and `println` builtins, which go to standard error. An unrecovered panic prints `panic: <value>` and exits with status 2, as Go does.

## What is rejected

`goalchemy check` reports unsupported constructs with stable codes (see [diagnostics](../specs/diagnostics.md)) and a one-line remedy. Release 0.1 rejects the following:

- generics, floating-point and complex numbers, and `uintptr`
- `unsafe`, cgo, `goto`, and range over functions
- pointers to slice or array elements, including pointer-receiver calls on elements
- concurrency under the default sequential gate (compile with `-gate cooperative`)
- imports other than module source and `goalchemy/lib/...` packages, including the standard library

Capabilities come from Goalchemy's library, imported under `goalchemy/lib/`. The packages keep the standard names, so code reads as ordinary Go:

| Import | Provides |
| --- | --- |
| `goalchemy/lib/errors` | `New`, `Is`, `Unwrap` |
| `goalchemy/lib/sync` | `Mutex`, `WaitGroup` (cooperative gate) |
| `goalchemy/lib/context` | `Context`, `CancelFunc`, `Background`, `WithCancel`, `WithTimeout`, `Canceled`, `DeadlineExceeded`, and the `Done` and `Err` methods (cooperative gate) |
| `goalchemy/lib/time` | `Duration`, its unit constants, and `Sleep` (cooperative gate) |
| `goalchemy/lib/runtime` | `Gosched` (cooperative gate) |
| `goalchemy/lib/task` | `All` (cooperative gate) |

Each package is ordinary Go that wraps the standard library, so programs still build and run with the Go toolchain. Importing a standard package such as `"sync"` directly is rejected with a remedy naming its `goalchemy/lib` replacement; `errors.As`, `errors.Join`, and `fmt` are not available.

## Cooperative execution

With `-gate cooperative` (or `gate: cooperative` in `goalchemy.yaml`), programs may use `go`, channels, `select`, and the synchronization capabilities above. Only one task runs at a time:

- Runnable tasks run in FIFO order. A task keeps control until it blocks, yields, returns, or panics.
- `select` chooses among ready cases with a seeded xorshift32 source: `GOALCHEMY_SEED`, default 1.
- `time.Sleep` and context deadlines use a virtual clock, which advances only when every task is blocked.
- A deadlock prints `fatal error: all goroutines are asleep - deadlock!` and exits with status 2.

Every target follows the same scheduler contract and the same shared lowering:

1. **Effect analysis** finds the functions that may suspend: those that use channels, `select`, or suspending capabilities, directly or through calls, function values, or interface methods.
2. **The shared IR pass** cuts each of those functions at its pause points into continuation blocks. The function becomes a resumable *frame*: an object holding its locals and the block to resume at.
3. **Each runtime** provides pause primitives (`recv`, `send`, `select`, lock, sleep, spawn) and a trampoline scheduler. Deferred calls, `panic`, and `recover` in frames are managed by the runtime, per task.

No target relies on native coroutines or threads for source tasks.

`goalchemy/lib/task` provides `task.All(fns ...func())`. It runs each function as a task, in argument order and one at a time, and returns when all have finished. With the Go toolchain the same package runs the functions as goroutines.

## Behavior Go leaves open

Goalchemy fixes some behavior that Go leaves to the implementation. These choices are identical on every target:

- Map iteration follows insertion order. An entry deleted before it is visited is skipped, and entries inserted during iteration are not visited.
- When `append` exceeds capacity, the new capacity is `max(needed, max(1, 2*cap))`.
- `[]byte(s)` and `[]rune(s)` have capacity equal to their length.
- Operands are evaluated left to right wherever Go allows a choice.

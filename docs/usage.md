# Using Goalchemy

Goalchemy compiles programs written in a restricted subset of Go into other languages. Source files are ordinary `.go` files: they build and run with the Go toolchain, and Goalchemy accepts the subset described in [the language specification](../specs/language.md). Release 0.1 supports the sequential and cooperative language on seven targets: lowered Go, TypeScript for Node.js, Python 3.10+, Java 21, C# for .NET 8, Rust (edition 2021), and C17. Run `goalchemy features` for the current feature matrix.

## Install

```sh
go build -o bin/goalchemy ./cmd/goalchemy
```

The repository's `go.mod` selects the Go 1.27.1 reference toolchain. Source programs must declare `go 1.25` or earlier. TypeScript output needs Node.js 22.6 or later, because it runs `.ts` files directly through Node's type stripping.

## Commands

| Command | Purpose |
| --- | --- |
| `goalchemy check [-gate g] [-tags t] [-json] [packages]` | Load, type-check, and validate against the language gate. |
| `goalchemy compile -target <go\|typescript\|python\|java\|csharp\|rust\|c\|ir> -out <dir> [packages]` | Write a complete, runnable target directory. |
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
  c: {out: out/c}
```

The file uses the same restricted YAML as the contract catalog: no anchors, aliases, or floats, and no unknown keys.

## Output

Each target directory contains the generated program, the runtime files it needs (one file per runtime function, plus shared representation files), a `README.md`, and `goalchemy.manifest.json`. The manifest records the source profile, compiler version, contract IDs, versions, and hashes, and every runtime and generated file. Output is deterministic for the same inputs.

- **Go**: `main.go`, `go.mod`, `rt/`. Run with `go run .`. Line directives map positions back to the Goalchemy source.
- **TypeScript**: sequential output has `main.ts` and `main.ts.map`; cooperative output adds portable `program.ts`, `program.ts.map`, and `host.ts`, with `main.ts` as its Node wrapper. Both include `rt/` and `package.json`. Run with `node main.ts`, and add `--enable-source-maps` for source positions in stack traces. Cooperative `host.ts` exports Promise-based `runHost(host)` with real monotonic time and an explicit portable output/failure adapter; see [the bounded lifecycle and executable-global limits](typescript-host-operations.md).
- **Python**: `main.py`, `rt/` (a package), and `main.py.lines`, which maps generated lines to Goalchemy source positions. Run with `python3 main.py`; Python 3.10 or later is required. Cooperative output exposes synchronous `main.runHost()` with monotonic time, a calling-thread owner and cleanup-before-return; see [its lifecycle and recursion limits](python-host-operations.md).
- **Java**: `Main.java`, `rt/`, `run.sh`, and `Main.java.lines` (generated lines to source positions). Run with `sh run.sh`, which compiles with `javac` and runs on a Java 21 or later JDK (`JAVA_HOME` is honored). Cooperative output also exposes `Main.runHost()` for an explicit serialized monotonic executable drive that returns after cleanup; see [its lifecycle and executable-global limits](java-host-operations.md).
- **C#**: `Main.cs`, `rt/`, `main.csproj`, `run.sh`, and `Main.cs.lines`. Run with `sh run.sh`, which compiles with the .NET 8 SDK's C# compiler and runs on .NET 8 (`DOTNET_ROOT` is honored); `dotnet run` also works. Cooperative output exposes Task-returning `GoProgram.runHost()` using a dedicated monotonic owner driver and serialized executable globals; see [its lifecycle and managed-recursion limits](csharp-host-operations.md).
- **Rust**: `src/main.rs`, `src/rt/`, `Cargo.toml`, `run.sh`, and `src/main.rs.lines`. Run std-only output with `sh run.sh` (plain `rustc`) or `cargo run --release`; native capability output builds with Cargo and maintained dependencies. The SDK package helper supplies its full dependency lock. Cooperative output also exposes `run_host() -> Result<(), HostError>`, using a serialized dedicated monotonic owner and cleanup-before-return; see [Rust lifecycle and stack limits](rust-host-operations.md). Values live in a traced heap collected at safepoints; set `GOALCHEMY_HEAP_STATS=1` to print heap statistics at exit and `GOALCHEMY_GC_THRESHOLD=<n>` to collect more often.
- **C**: `main.c`, `rt/` (`gx.h` and one `.c` file per runtime function), `run.sh`, and `main.c.lines`. Run with `sh run.sh`, which builds with `cc -std=c17` and links the Boehm-Demers-Weiser collector (bdwgc 8.x with threads): `GOALCHEMY_BDWGC` may name an install prefix, otherwise `pkg-config bdw-gc` or `-lgc` is used. `CC` and `CFLAGS` are honored, so `CC=clang CFLAGS='-fsanitize=address,undefined'` builds a sanitized program.

### Libraries

Non-`main` packages can emit value libraries for Go, TypeScript, Java, C#, Python, Rust and C:

- [Go libraries](go-library-boundary.md) expose copied values and cancellable, serialized calls.
- [TypeScript libraries](typescript-library-boundary.md) emit portable JavaScript and declarations for Node and browsers.
- [Java libraries](java-library-boundary.md) emit `Generated.java`, `build.sh` and a named-package JAR with byte arrays, typed values and cancellable asynchronous operations. Production crypto requires the pinned BC 1.86 dependency alongside JDK 21.
- [C# libraries](csharp-library-boundary.md) emit .NET 8 class libraries with owned values and cancellable asynchronous operations. Crypto and HTTP use .NET built-ins without NuGet dependencies.

Python libraries emit an importable generated module with owned byte/value
boundaries and cancellable sync/async operations. Production crypto uses
maintained cryptography; HTTP uses the standard library. See the adjacent
[Python TDF3 package and boundary evidence](../../sdk/docs/generated-python-library.md).
[Rust libraries](rust-library-boundary.md) emit owned typed Cargo Result/Future APIs
with serialized source owners and cleanup-before-publication. Native crypto/HTTP
use maintained locked crates; see [Rust TDF3 delivery](../../sdk/docs/generated-rust-library.md).
Unsupported public value shapes on supported value-library targets fail with `GCE007`.

C aggregate or suspending exports emit `goalchemy.h`, `main.c`, `rt/` and
`build.sh`, producing `libgoalchemy.a`. The owned `gxc_value` API recursively
copies inputs and detaches outputs/errors, preserving explicit byte lengths and
exact integer bits. Calls serialize on a registered source owner with fresh
initialization; consumers need no collector initialization. Native callbacks
use owned data and cancellation signals; completion waits for actual cleanup
and owner joins. Production capabilities require threaded Boehm, OpenSSL 3 and
libcurl. The adjacent [C TDF3 delivery](../../sdk/docs/generated-c-library.md)
documents typed headers, submit/drive/wake/cancel/take/destroy, buffer releases,
limits and native consumer evidence.

For sequential scalar/string exports, compiling a package other than `main` for the C target produces `goalchemy.h`, `main.c`, `rt/`, and `build.sh`, which builds `libgoalchemy.a`. The host calls `goalchemy_init()` once, then the exported functions of the root package. Each export is `int pkg_Name(params..., results*...)`: it returns 0 on success and 1 after an unrecovered panic, whose report `goalchemy_panic_message()` returns. Integers cross as `int64_t` (`uint64_t` for `uint64` and `uint`), Booleans as `bool`, and strings as a pointer and a length. Exported functions must not suspend, and other parameter types are rejected.

When the working directory or a parent has a `.toolchains` directory holding `jdk-*` or `dotnet`, `goalchemy run` and the test suites use those toolchains.
On Linux x64, `scripts/fetch-toolchains.sh` installs the versions and verified
archives in `toolchains.lock`, including bdwgc and the clang used by
`tests/sanitize`. The sanitizer runner uses that clang with normal address
randomization. `scripts/fetch-toolchains.sh llvm` installs only clang and its
private `libtinfo5` dependency.

Programs write output with Go's `print` and `println` builtins, which go to standard error. An unrecovered panic prints `panic: <value>` and exits with status 2, as Go does.

## What is rejected

`goalchemy check` reports unsupported constructs with stable codes (see [diagnostics](../specs/diagnostics.md)) and a one-line remedy. Release 0.1 rejects the following:

- generics, floating-point and complex numbers, and `uintptr`
- `unsafe`, cgo, `goto`, and range over functions
- pointers to slice or array elements, including pointer-receiver calls on elements
- concurrency under the default sequential gate (compile with `-gate cooperative`)
- imports other than module source and `github.com/eugenioenko/goalchemy/lib/...` packages, including the standard library

Capabilities come from Goalchemy's library, imported under `github.com/eugenioenko/goalchemy/lib/`. The packages keep the standard names, so code reads as ordinary Go:

| Import | Provides |
| --- | --- |
| `github.com/eugenioenko/goalchemy/lib/errors` | `New`, `Is`, `Unwrap` |
| `github.com/eugenioenko/goalchemy/lib/sync` | `Mutex`, `WaitGroup` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/context` | `Context`, `CancelFunc`, `Background`, `WithCancel`, `WithTimeout`, `Canceled`, `DeadlineExceeded`, and the `Done` and `Err` methods (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/time` | `Duration`, its unit constants, and `Sleep` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/runtime` | `Gosched` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/task` | `All` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/crypto` | `Key`, SHA-256, HMAC-SHA256, HKDF-SHA256, AES-256-GCM, RSA-2048 OAEP, RS256, ES256, P-256 ECDH, key generation, PEM and JWK |
| `github.com/eugenioenko/goalchemy/lib/encoding` | `Base64Encode`, `Base64Decode`, `Base64URLEncode`, `Base64URLDecode` |
| `github.com/eugenioenko/goalchemy/lib/http` | `Do`, a bounded GET or POST exchange |
| `github.com/eugenioenko/goalchemy/lib/clock` | `Unix`, host wall-clock seconds |
| `github.com/eugenioenko/goalchemy/lib/callback` | `Request`, a bounded call to a host-registered callback |

The [runtime library reference](library.md) documents every function: its Go signature, gate, bounds, error behavior, per-target availability and the native dependencies of each target. It is generated from the contracts with `make spec-generate`.

Each package is ordinary Go that wraps the standard library, so programs still build and run with the Go toolchain. Importing a standard package such as `"sync"` directly is rejected with a remedy naming its `github.com/eugenioenko/goalchemy/lib` replacement; `errors.As`, `errors.Join`, and `fmt` are not available.

## Cooperative execution

With `-gate cooperative` (or `gate: cooperative` in `goalchemy.yaml`), programs may use `go`, channels, `select`, and the synchronization capabilities above. Only one task runs at a time:

- Runnable tasks run in FIFO order. A task keeps control until it blocks, yields, returns, or panics.
- `select` chooses among ready cases with a seeded xorshift32 source: `GOALCHEMY_SEED`, default 1.
- The default entry uses a virtual clock for `time.Sleep` and context deadlines,
  advancing only when every task is blocked. Generated Go programs using HTTP
  select an explicit host entry with owner-local monotonic real time. TypeScript
  cooperative output can select real time through portable `host.ts`/`runHost`;
  its module-global source state is not an isolated library instance. Java
  cooperative output exposes `Main.runHost()` with the same executable-global
  limitation and `System.nanoTime` deadlines. C# cooperative output exposes
  Task-returning `GoProgram.runHost()` with `Stopwatch` deadlines and serialized
  executable globals; it is not an isolated library API. Python cooperative
  output exposes blocking `main.runHost()` with `monotonic_ns` deadlines, native
  mailbox completion and serialized executable globals; it is not an SDK library.
  C cooperative output exposes `goalchemy_run_host()` with checked monotonic
  deadlines, owned native wire records and resource ACK before return; see the
  [C executable boundary and limits](c-host-operations.md).
- A deadlock prints `fatal error: all goroutines are asleep - deadlock!` and exits with status 2.

Every target follows the same scheduler contract and the same shared lowering:

1. **Effect analysis** finds the functions that may suspend: those that use channels, `select`, or suspending capabilities, directly or through calls, function values, or interface methods.
2. **The shared IR pass** cuts each of those functions at its pause points into continuation blocks. The function becomes a resumable *frame*: an object holding its locals and the block to resume at.
3. **Each runtime** provides pause primitives (`recv`, `send`, `select`, lock, sleep, spawn) and a trampoline scheduler. Deferred calls, `panic`, and `recover` in frames are managed by the runtime, per task.

No target relies on native coroutines or threads for source tasks.

Generated Go `lib/http.Do` runs the bounded native transport on host workers.
Workers enqueue owned completions; the scheduler alone resumes source tasks,
so concurrent source cancellation remains runnable during network and body
reads. Context deadlines start at creation and an earlier parent deadline
shortens the required real HTTP timeout. Entry return cancels and cleans pending
background operations. Other targets still reject this capability until their
adapters are implemented; exported asynchronous libraries remain gated. See
[host lifecycle and target-port requirements](host-operations.md).

`github.com/eugenioenko/goalchemy/lib/task` provides `task.All(fns ...func())`. It runs each function as a task, in argument order and one at a time, and returns when all have finished. With the Go toolchain the same package runs the functions as goroutines.

## Behavior Go leaves open

Goalchemy fixes some behavior that Go leaves to the implementation. These choices are identical on every target:

- Map iteration follows insertion order. An entry deleted before it is visited is skipped, and entries inserted during iteration are not visited.
- When `append` exceeds capacity, the new capacity is `max(needed, max(1, 2*cap))`.
- `[]byte(s)` and `[]rune(s)` have capacity equal to their length.
- Operands are evaluated left to right wherever Go allows a choice.

## Byte storage

TypeScript byte slices and arrays use `Uint8Array`, including named types
with underlying `uint8` elements. Slice views share native storage; byte
string conversions copy and preserve arbitrary Go string bytes. Java byte
slices and arrays use native `byte[]`, with unsigned reads through
Go's existing `long` integer representation. Java preserves aliases, zeroed
capacity and arbitrary-byte string copies too. C# byte slices and arrays use
native `byte[]`, with primitive reads widened to Go `long`, zeroed capacity,
shared views and independent arbitrary-byte string copies. Python byte slices
and arrays use native `bytearray`, with fixed-length shared views, zeroed
capacity and independent conversions to/from immutable `bytes` Go strings.
Rust byte slices and arrays use traced native `Vec<u8>` backing, with typed nil
headers, zeroed capacity, shared views and independent binary string copies.
C byte slices and arrays use collector-owned native `uint8_t` backing, with
typed nil headers, zeroed capacity, shared interior views and independent
binary string copies. See the bounded [TypeScript evidence](typescript-byte-storage.md),
[Java evidence](java-byte-storage.md), [C# evidence](csharp-byte-storage.md)
[Python evidence](python-byte-storage.md), [Rust evidence](rust-byte-storage.md)
and [C evidence](c-byte-storage.md)
for coverage and remaining SDK, browser and library requirements.

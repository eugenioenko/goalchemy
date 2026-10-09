# Using Goalchemy

Goalchemy compiles programs written in a restricted subset of Go into other languages. Source files are ordinary `.go` files: they build and run with the Go toolchain, and Goalchemy accepts the subset described in [the language specification](../specs/language.md). Targets are lowered Go, TypeScript for Node.js, Python 3.10+, Java 21, C# for .NET 8, Rust (edition 2021), C17, and experimental Swift 6.4 on Linux x86_64. Run `goalchemy features` for the current feature matrix.

## Install

```sh
go build -o bin/goalchemy ./cmd/goalchemy
```

The repository's `go.mod` selects the Go 1.27.1 reference toolchain. Source programs must declare `go 1.25` or earlier. TypeScript output needs Node.js 22.6 or later, because it runs `.ts` files directly through Node's type stripping.

## Commands

| Command | Purpose |
| --- | --- |
| `goalchemy check [-gate g] [-tags t] [-json] [packages]` | Load, type-check, and validate against the language gate. |
| `goalchemy compile -target <go\|typescript\|python\|java\|csharp\|rust\|c\|swift\|ir> -out <dir> [packages]` | Write a complete, runnable target directory. |
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
compact_names: false        # readable private identifiers (default)
targets:
  go: {out: out/go}
  typescript: {out: out/ts}
  python: {out: out/py}
  java: {out: out/java}
  csharp: {out: out/cs}
  rust: {out: out/rs}
  c: {out: out/c}
  swift: {out: out/swift}
```

The file uses the same restricted YAML as the contract catalog: no anchors, aliases, or floats, and no unknown keys.

`compile`, `run`, and `build` accept `--compact-names`. `compile` and `run`
default to readable names and do not load `goalchemy.yaml`. `build` uses the
configuration's `compact_names` setting unless the flag is explicitly present:
`--compact-names` selects compact names and `--compact-names=false` selects
readable names.

```sh
goalchemy compile -target rust -out out/rust --compact-names .
goalchemy run -target python --compact-names .
goalchemy build --compact-names=false
```

## Generated names

Generated private names are readable by default for Go, TypeScript, Python,
Java, C#, Rust, C, and Swift. Source package, type, function, global, local, and field
names supply identifier stems. Category prefixes and deterministic numeric
suffixes distinguish reserved words, shadowed locals, and repeated names across
packages. Unicode characters are encoded in ASCII as `_u` followed by their
hexadecimal code point and `_`.

For example, an internal function for `main.Calculate` may be named
`fn_main_Calculate_2`; a source local `count` may be `v_count_7`. Targets use
their own legal separators and type prefixes. A named Go struct contributes its
package/type stem to the generated representation; when several named types
share an underlying representation, the lexicographically smallest source name
is chosen deterministically. Anonymous aggregates use shape names such as
`struct`, `array`, or `map`. Unnamed parameters, results, captures, and
temporaries use `param`, `result`, `capture`, and `temp` stems.

`--compact-names` replaces compiler-private names with shorter identifiers
derived from the same IR identities. Public library function/type/field names,
capability symbols, runtime type strings, and serialized identities retain
their established contracts in both modes. Native Go embedding and field rules
also remain intact. Go boundary values from dependency packages receive stable
exported `Source_<qualified-type>_<ID>` aliases, including values nested in
fields and collections.

Public façade names cannot occupy names reserved by their target's generated
API. These collisions produce a library-boundary diagnostic in both modes,
rather than invalid target code. For example, Rust rejects public structs named
`V`, `LibraryError`, or `Vec`, and exported functions named `Some`, `None`, `Ok`,
or `Err`; Python rejects exported functions named `None`, `True`, or `False`.
These names remain usable internally when the source language permits them.

Readable names help inspect generated source; they do not reconstruct Go's
original control flow or replace source maps. Some targets still store values
in runtime frames or numeric aggregate slots and identify them through named
helpers or slot constants. Runtime support has its own naming conventions, and
IR dumps keep their diagnostic identifiers. Internal names are not a stable
API across compiler versions. Neither naming mode promises a performance
improvement.

## Initialization guidance

Prefer explicit constructors or setup functions for configuration, I/O, key
generation and resource acquisition. They make dependencies, errors and cleanup
visible to callers. Avoid hiding this work in `init()` or in package-level
variables initialized by calls to setup functions.

Small deterministic initialization remains supported. Goalchemy preserves Go's
package-variable dependency ordering and `init()` functions in its central
initialization routine. Splitting generated code into files does not remove
those effects.

The SDK-capable library boundary creates fresh source state and runs package
initialization once per operation. An expensive initializer or externally
visible side effect therefore repeats across calls. This differs from a native
Go process, which initializes its packages once at startup. Consult the target's
library-boundary documentation for its lifecycle; the legacy sequential C
scalar/string API uses an explicit `goalchemy_init()` call.

## Output

Each target directory contains the generated program, the runtime files it needs (one file per runtime function, plus shared representation files), a `README.md`, and `goalchemy.manifest.json`. The manifest records the source profile, compiler version, contract IDs, versions, and hashes, and every runtime and generated file. Output is deterministic for the same inputs.

Program declarations are grouped by their original Go package. The manifest's
`source_packages` records each included package's import path, dependencies and
native source files. Shared representations and the central initializer remain
separate from package-owned declarations. Native public entry points and library
imports stay the same; this does not publish a separate native library for every
Go package. Existing identifier prefixes and compact-name behavior are retained.

Each source file has its own diagnostic map where source positions are available.
When an output directory is reused successfully, Goalchemy removes obsolete files
listed by its previous generated/runtime inventory and preserves unlisted caller
files. Failed emission does not clean that inventory. IR-only output does not use
native output inventories.

- **Go**: `main.go`, `shared.go`, `pkg_*.go`, `go.mod`, `rt/`, all in the existing native package. Run with `go run .`. Line directives map positions back to the Goalchemy source.
- **TypeScript**: native `pkg_*.ts` ESM modules and leaf `shared.ts`; sequential output has `main.ts`, while cooperative output adds portable `program.ts` and `host.ts`, with `main.ts` as its Node wrapper. Source files have individual `.ts.map` files. Both include `rt/` and `package.json`. Run with `node main.ts`, and add `--enable-source-maps` for source positions in stack traces. Cooperative `host.ts` exports Promise-based `runHost(host)` with real monotonic time and an explicit portable output/failure adapter; see [the bounded lifecycle and executable-global limits](typescript-host-operations.md).
- **Python**: `main.py`, native `pkg_*.py` modules, leaf `_shared.py`, and `rt/` (a package). Individual `.py.lines` sidecars map generated lines to Goalchemy source positions. Run with `python3 main.py`; Python 3.10 or later is required. Cooperative output exposes synchronous `main.runHost()` with monotonic time, a calling-thread owner and cleanup-before-return; see [its lifecycle and recursion limits](python-host-operations.md).
- **Java**: `Main.java`, source-package `pkg_*.java` holders, leaf `_GoalchemySupport.java`, `rt/`, `run.sh`, and per-file `.java.lines` sidecars (generated lines to source positions). Run with `sh run.sh`, which compiles the native source inventory with `javac` and runs on a Java 21 or later JDK (`JAVA_HOME` is honored). Cooperative output also exposes `Main.runHost()` for an explicit serialized monotonic executable drive that returns after cleanup; see [its lifecycle and executable-global limits](java-host-operations.md).
- **C#**: `Main.cs`, source-package `pkg_*.cs` files and `shared.cs` declaring one native partial `GoProgram`, `rt/`, `main.csproj`, `run.sh`, and per-file `.cs.lines` sidecars. Run with `sh run.sh`, which compiles the generated/runtime inventory with the .NET 8 SDK's C# compiler (`DOTNET_ROOT` is honored); `dotnet run` also works. The project includes additional root source adapters. Cooperative output exposes Task-returning `GoProgram.runHost()` using a dedicated monotonic owner driver and serialized executable globals; see [its lifecycle and managed-recursion limits](csharp-host-operations.md).
- **Rust**: `src/main.rs` (or `src/lib.rs` for libraries), native `src/pkg_*.rs` modules, `src/shared.rs`, `src/rt/`, `Cargo.toml`, `run.sh`, and per-file `.rs.lines` sidecars. Explicit module paths allow SDK adapters to rename the generated entry file. Run std-only output with `sh run.sh` (plain `rustc`) or `cargo run --release`; native capability output builds with Cargo and maintained dependencies. Cargo compiles explicit native targets; the SDK package helper supplies its full dependency lock. Cooperative output also exposes `run_host() -> Result<(), HostError>`, using a serialized dedicated monotonic owner and cleanup-before-return; see [Rust lifecycle and stack limits](rust-host-operations.md). Values live in a traced heap collected at safepoints; set `GOALCHEMY_HEAP_STATS=1` to print heap statistics at exit and `GOALCHEMY_GC_THRESHOLD=<n>` to collect more often.
- **C**: `main.c`, package-owned `pkg_*.c` translation units, `shared.c`, declaration-only `goalchemy_internal.h`, `rt/` (`gx.h` and one `.c` file per runtime function), `run.sh`, and per-file `.c.lines` sidecars. The scripts compile the explicit generated/runtime inventory as separate native units. Run with `sh run.sh`, which builds with `cc -std=c17` and links the Boehm-Demers-Weiser collector (bdwgc 8.x with threads): `GOALCHEMY_BDWGC` may name an install prefix, otherwise `pkg-config bdw-gc` or `-lgc` is used. Programs that use `lib/crypto` or `lib/http` also link OpenSSL (`-lssl -lcrypto`) and, for HTTP, libcurl through `pkg-config libcurl`. `CC`, `CFLAGS` and `LDLIBS` are honored, so `CC=clang CFLAGS='-fsanitize=address,undefined'` builds a sanitized program.
- **Swift**: `main.swift` (or `Generated.swift` for libraries), package-owned `pkg_*.swift` files, `shared.swift` with the canonical type table, `rt/`, `run.sh`, `LICENSE`, and per-file `.swift.lines` sidecars. Native scripts and SwiftPM explicitly list sources within one native module. Run with `sh run.sh`, which builds a native C module and Swift executable. Requires Swift 6.4, OpenSSL 3, zlib, libcurl and pkg-config even for pure-logic programs. Linux x86_64 is the verified baseline; see [Swift setup, semantics and public libraries](swift-target.md).

### Libraries

Non-`main` packages can emit value libraries for Go, TypeScript, Java, C#, Python, Rust, C and Swift:

- [Go libraries](go-library-boundary.md) expose copied values and cancellable, serialized calls.
- [TypeScript libraries](typescript-library-boundary.md) emit portable JavaScript and declarations for Node and browsers.
- [Java libraries](java-library-boundary.md) emit `Generated.java`, `build.sh` and a named-package JAR with byte arrays, typed values and cancellable asynchronous operations. Production crypto requires the pinned BC 1.86 dependency alongside JDK 21.
- [C# libraries](csharp-library-boundary.md) emit .NET 8 class libraries with owned values and cancellable asynchronous operations. Crypto and HTTP use .NET built-ins; IEEE CRC32 adds the official Microsoft `System.IO.Hashing` 8.0.0 NuGet package.
- [Swift libraries](swift-target.md) emit a `GoalchemyGenerated` SwiftPM product with typed values, lossless Go strings, and cancellable `wait()`/async `value()` calls. All current Swift packages need OpenSSL 3, zlib and libcurl development files.

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
- imports other than module source and `github.com/eugenioenko/goalchemy/std/...` or `.../lib/...` packages, including the standard library

Goalchemy replaces the standard library with packages from two roots. Both keep the standard names, so code reads as ordinary Go:

- **`github.com/eugenioenko/goalchemy/std/...`** holds pure-logic packages written once in the Goalchemy subset. The compiler translates them with your program, so they behave identically on every target and need no native dependencies.
- **`github.com/eugenioenko/goalchemy/lib/...`** holds capability packages that reach the host. Each has a native implementation per target, checked against a contract, and some need native dependencies.

The split, and the core runtime layer behind both, is explained in [the runtime library reference](library.md#three-layers).

| Import | Provides |
| --- | --- |
| `github.com/eugenioenko/goalchemy/std/strings` | `Builder`, `Split`, `Join`, `Fields`, `Index`, `Contains`, `Trim*`, `Cut`, `Replace`, `NewReplacer`, `ToLower`/`ToUpper`, `EqualFold`, and more |
| `github.com/eugenioenko/goalchemy/std/strconv` | `Itoa`, `Atoi`, `ParseInt`/`ParseUint`/`ParseBool`, `FormatInt`/`FormatUint`/`FormatBool`, `Quote`, `NumError` |
| `github.com/eugenioenko/goalchemy/std/bytes` | `Buffer`, `Equal`, `Compare`, `Index`, `Split`, `Fields`, `TrimSpace`, and more |
| `github.com/eugenioenko/goalchemy/std/sort` | `Sort`, `Stable`, `Ints`, `Strings`, `Search`, `Reverse` over `Len`/`Less`/`Swap` |
| `github.com/eugenioenko/goalchemy/std/unicode` | `IsLetter`, `IsDigit`, `IsSpace`, `IsUpper`, `IsPrint`, `ToLower`, `ToUpper`, `SimpleFold`, and more |
| `github.com/eugenioenko/goalchemy/std/unicode/utf8` | `DecodeRune`, `EncodeRune`, `AppendRune`, `RuneCountInString`, `ValidString`, and more |
| `github.com/eugenioenko/goalchemy/std/encoding/hex` | `EncodeToString`, `DecodeString`, `Encode`, `Decode` |
| `github.com/eugenioenko/goalchemy/std/encoding/binary` | `BigEndian`/`LittleEndian` `Uint16/32/64`, `PutUint*`, `AppendUint*`, varints |

| `github.com/eugenioenko/goalchemy/lib/errors` | `New`, `Is`, `Unwrap` |
| `github.com/eugenioenko/goalchemy/lib/sync` | `Mutex`, `WaitGroup` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/context` | `Context`, `CancelFunc`, `Background`, `WithCancel`, `WithTimeout`, `Canceled`, `DeadlineExceeded`, and the `Done` and `Err` methods (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/time` | `Duration`, its unit constants, and `Sleep` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/runtime` | `Gosched` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/task` | `All` (cooperative gate) |
| `github.com/eugenioenko/goalchemy/lib/crypto` | `Key`, SHA-256, HMAC-SHA256, HKDF-SHA256, AES-256-GCM, RSA OAEP (2048, 3072 and 4096-bit), RS256/384/512, ES256/384/512, P-256/P-384/P-521 ECDH, key generation, PEM and JWK |
| `github.com/eugenioenko/goalchemy/lib/encoding` | `Base64Encode`, `Base64Decode`, `Base64URLEncode`, `Base64URLDecode` |
| `github.com/eugenioenko/goalchemy/lib/http` | `Do`, a bounded GET or POST exchange |
| `github.com/eugenioenko/goalchemy/lib/clock` | `Unix`, host wall-clock seconds |
| `github.com/eugenioenko/goalchemy/lib/callback` | `Request`, a bounded call to a host-registered callback |

The [runtime library reference](library.md) documents every function. For `lib/` it lists the gate, bounds, error behavior, per-target availability and native dependencies. It is generated from the `std/` and `lib/` sources and the contracts with `make spec-generate`.

Every package is ordinary Go, so programs still build and run with the Go toolchain. Importing a standard package such as `"sync"` or `"strings"` directly is rejected with a remedy naming its `std/` or `lib/` replacement; `errors.As`, `errors.Join`, and `fmt` are not available.

## Cooperative execution

With `-gate cooperative` (or `gate: cooperative` in `goalchemy.yaml`), programs may use `go`, channels, `select`, and the synchronization capabilities above. Only one task runs at a time:

- Runnable tasks run in FIFO order. A task keeps control until it blocks, yields, returns, or panics.
- `select` chooses among ready cases with a seeded xorshift32 source: `GOALCHEMY_SEED`, default 1.
- The default entry uses a virtual clock for `time.Sleep` and context deadlines,
  advancing only when every task is blocked. Generated programs that use
  `lib/crypto`, `lib/http` or `lib/callback` start on their target's host entry
  instead, with owner-local monotonic real time. TypeScript
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
background operations. Other targets run `lib/http.Do` through their own host
adapters; exported asynchronous libraries remain gated. See
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

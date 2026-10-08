# IEEE CRC32 host mappings

`lib/checksum.CRC32IEEE` computes the ZIP/IEEE CRC-32 of the logical byte slice.
Nil and empty inputs return zero; bytes outside its length are ignored, inputs
stay unchanged, and results retain all unsigned 32 bits. It is a checksum and
provides no cryptographic authentication.

| Target/entry | Implementation | Dependency |
| --- | --- | --- |
| Go | `hash/crc32.ChecksumIEEE` | Standard library |
| Java | `java.util.zip.CRC32` | JDK standard library |
| Python | `zlib.crc32` | Standard library |
| TypeScript Node executable, conformance, or Node library entry | `node:zlib.crc32` | Node standard library |
| C# | `System.IO.Hashing.Crc32.HashToUInt32(ReadOnlySpan<byte>)` | Official Microsoft NuGet package, pinned to 8.0.0 |
| TypeScript browser or portable library entry | Slicing-by-8 IEEE implementation | No standard browser CRC API |
| C | Slicing-by-8 IEEE implementation | No standard C CRC API |
| Rust | `crc32fast::hash` with runtime CPU detection | Cargo crate, pinned to 1.5.2 |
| Swift | `crc32` via the native zlib shim | System zlib; Swift has no standard CRC32 API |

Host APIs choose their own native implementation, vectorization, and hardware
acceleration. Delegation permits host optimizations; Goalchemy does not guarantee
that any particular CPU instruction or accelerated path runs.

## TypeScript entries

Generated executables and the conformance harness install the Node checksum
adapter automatically. Generated libraries expose conditional package exports:
Node selects `dist/node.js`, and browser/default consumers select `dist/main.js`.
Import the installed package by name for automatic selection, or use
`dist/node.js` directly in Node. Direct `dist/main.js` imports intentionally
select the portable implementation, including when imported in Node.
The Node library entry installs only the checksum adapter; it does not install
executable stdout, stderr, seed, or process-exit behavior. Portable dependency
graphs contain no Node imports, `Buffer`, or `process` references.

The supported Node minimum remains 22.6; the standard
[`zlib.crc32` API](https://nodejs.org/download/release/v22.18.0/docs/api/zlib.html#zlibcrc32data-value)
has existed since 22.2. Calls pass a `Uint8Array` view of the logical bytes,
including the backing view's byte offset; they never encode bytes as text.

## C# dependency and consumers

[`System.IO.Hashing` 8.0.0](https://www.nuget.org/packages/System.IO.Hashing/8.0.0)
is a first-party Microsoft package, separate from the shared .NET 8 runtime.
CRC-using generated projects include an exact `PackageReference`,
`packages.lock.json` with its NuGet content hash, and locked restore. `run.sh`
keeps the direct C# compiler path, restores the package, references its .NET 8
DLL, and copies that DLL beside the executable. `dotnet run --project main.csproj`
uses normal SDK dependency resolution. Programs that do not link CRC keep
their existing build script and have no checksum package dependency.

For a generated library, use a `ProjectReference` to `main.csproj` so normal
NuGet/SDK resolution carries the package transitively into the consumer's output.
Consumers that reference only `lib/main.dll` must also reference
`System.IO.Hashing` 8.0.0 and deploy its DLL. `build.sh` produces the generated
library; deployment of an assembly alone does not install its dependencies.

Repository conformance and tests compile the complete C# runtime, so their
builds require the package even when an individual test exercises another
capability. `scripts/ci-bootstrap.sh` prewarms the locked checksum project;
`targets/csharp/tests/checksum-dependencies.sh` repeats locked restore and
returns the package DLL used by direct compiler harnesses.

## Rust dependency

Generated native Cargo projects and the conformance harness pin
[`crc32fast` 1.5.2](https://github.com/srijs/rust-crc32fast). Its default `std`
feature remains enabled so it can detect CPU features and select an accelerated
IEEE CRC32 implementation when supported. The runtime hashes the logical byte
slice directly without copying or mutating its backing storage. Empty and nil
slices return zero. The harness lockfile records the crate checksum and CI
prefetches and builds that locked graph before running contracts.

Native Cargo output includes this direct dependency; programs that use only
subset logic keep their existing standard-library `rustc` path. Structural
standard-library-only byte and scheduler probes omit production checksum,
crypto, encoding, HTTP, and callback modules; CRC behavior is checked separately
through the Cargo harness and generated executable/library consumers.

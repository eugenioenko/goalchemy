# TypeScript byte storage evidence

This bounded Phase 4 change specializes TypeScript byte storage. It does not
complete host operations, library exports, browser SDK acceptance, or the
other non-Go targets.

The emitter recognizes the underlying `uint8` integer kind, including `byte`,
non-generic aliases, named elements and named slices. Byte arrays use generated
`Uint8Array` zero/clone/in-place-set helpers. Slice headers retain backing
identity, offset, length and capacity; a byte hint on the nil header selects
typed storage when appended to later. Nil remains distinct from allocated
empty storage. Capacity is physically allocated and zeroed for byte `make`
and append growth. The established capacity rule is unchanged.

Typed array `set` provides overlapping copy and in-capacity append semantics;
reallocation copies into separate storage. Clear only fills the viewed range.
Array assignment updates existing storage so slices of an array continue to
observe assignment; array value copies allocate separate storage. The byte
string conversions still copy and preserve one code unit per Go string byte,
including NUL, high bytes and invalid UTF-8. No text decoder is substituted.
These runtime modules introduce no Node imports or `Buffer` dependency.
Generic non-byte slices retain their previous representation and behavior.

Headers, arrays, interface values and closures remain ordinary GC-reachable
objects. There is no external ownership or manual lifetime. Permitted pointers
to arrays and struct fields retain their stable object identity. Element
addresses, slice-to-array pointer conversion and source generics remain
outside the supported language subset; this change does not expand it.

## Verification

Run from the Goalchemy repository root. The loader and native oracle keep
their pinned default toolchains; only the compiler tool itself is selected
with `GOTOOLCHAIN`.

```sh
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec validate
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec generate -check
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -count=1
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=typescript go test ./tests/language -count=1
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=go,typescript,python,java,csharp,rust,c FIXTURE=byte_storage go test ./tests/language -count=1
node --test targets/typescript/tests/byte_storage.test.ts
```

The language fixture compares actual generated program observations with
native Go: array/view aliases, full slices, growth detachment, overlapping
append/copy in both directions, clear, nil/empty, named/alias types, array
copies and slice-to-array conversion, struct fields and pointers,
interfaces/closures, uint8 overflow, byte strings and caught bounds panics.
The contract tests also inspect emitted byte-array helpers and nil/make paths,
and assert actual `Uint8Array` backing so boxed arrays cannot pass on matching
observations alone. TypeScript runtime typechecking requires installed `tsc`
and `targets/typescript/node_modules/@types/node`.

A stronger independent Go growth-zeroing test is target-specific while byte
backends are implemented. It remains visible and defaults to TypeScript;
subsequent workers can select their backend explicitly:

```sh
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -run TestTypeScriptByteGrowthNativeOracle -count=1
GOTOOLCHAIN=go1.25.14 GOALCHEMY_BYTE_TARGETS=c go test ./tests/contracts -run TestTypeScriptByteGrowthNativeOracle -count=1
```

The second command reproduces the remaining backends' preexisting gap:
reslicing freshly grown byte slices into spare capacity reads an unset value.
Python historically printed `None`; its subsequent
[native byte-storage assignment](python-byte-storage.md) fixed that failure
and runs a default growth test. Rust historically printed `nil`; its subsequent
[native byte-storage assignment](rust-byte-storage.md) fixed that failure and
runs a default growth test. At the original TypeScript baseline C printed `nil`; its subsequent
[native-byte checks](c-byte-storage.md) pass. At TypeScript acceptance C#
raised a host `NullReferenceException` and Java raised `NullPointerException`;
its subsequent [native byte-storage assignment](java-byte-storage.md) fixed
that regression and now runs its own default growth test. C# subsequently
fixed its recorded `NullReferenceException` with [native byte storage](csharp-byte-storage.md)
and also runs a default growth test. The test requires
zero, including named `uint8` elements and spread append growth. It is not
skipped or assigned a hand-written passing observation. Other backends need
their own full-capacity allocation and zeroing before opting into this test.
The existing generic non-byte TypeScript append path also leaves spare
capacity uninitialized; its repair remains a separate runtime follow-up.

## Memory and throughput

Exact reproduction, with a fresh Node process for each measurement:

```sh
node --expose-gc targets/typescript/tests/byte_storage_bench.ts typed 2
node --expose-gc targets/typescript/tests/byte_storage_bench.ts boxed 2
node --expose-gc targets/typescript/tests/byte_storage_bench.ts typed 8
node --expose-gc targets/typescript/tests/byte_storage_bench.ts boxed 8
```

Measured on Linux x86_64, kernel 6.8.0-87-generic, AMD Ryzen 7 6800H,
16 logical CPUs, Node v24.15.0. The script makes source and destination
buffers, fills the source, performs 16 copies and appends source to itself
into a new doubled-capacity buffer. All three buffers remain reachable
through checks after GC. Thus retained backing is four times the input size.
The boxed comparison calls the unchanged generic number-slice path with the
same values and operations. It includes the generic overlap temporary cost.

| Input | Storage | Exact typed backing | Heap delta after GC | Copy 16 times | Append once |
| --- | --- | --- | --- | --- | --- |
| 2 MiB | Uint8Array | 8,388,608 B | 11,896 B | 1.87 ms | 2.50 ms |
| 2 MiB | Generic array | — | 67,134,000 B | 449.66 ms | 80.70 ms |
| 8 MiB | Uint8Array | 33,554,432 B | 11,144 B | 13.57 ms | 12.00 ms |
| 8 MiB | Generic array | — | 268,459,984 B | 1,635.00 ms | 554.07 ms |

One-byte backing per retained element is checked directly through
`Uint8Array.byteLength`. Heap and timing results are illustrative observations,
not portable thresholds. The process's `arrayBuffers` delta can include
release of Node's prior TypeScript parser buffers, and RSS includes allocator
and process noise. Mandatory tests use structural backing assertions instead
of a noisy RSS expectation. Browser engines have not been benchmarked.
Allocations remain bounded by the existing 2^32−1-element host limit and
available host memory. Byte-string conversion creates immutable host strings
and bounded chunk temporaries; it is not a streaming interface.

## Remaining SDK requirements

Typed storage is usable by future portable Web Crypto/fetch adapters, but
none are implemented here. Crypto/HTTP unavailable target capabilities keep
reporting GCE002; unsupported library targets keep reporting GCE006. Host
operation scheduling, completion retention, cancellation, shutdown and
exported-library lifetimes remain Phase 4 work. Accurate package declarations,
promise APIs, adapters and independent Node and real browser interop remain
Phase 5 work. Java byte storage has subsequently passed its bounded checks;
C# byte storage has subsequently passed its bounded checks too. Python now
has [native byte-storage evidence](python-byte-storage.md), and Rust now has
[native byte-storage evidence](rust-byte-storage.md). C subsequently gained
[native byte-storage evidence](c-byte-storage.md).
See [the current Java evidence](java-byte-storage.md) and [C# evidence](csharp-byte-storage.md).

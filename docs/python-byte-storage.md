# Python byte storage evidence

This bounded Phase 4 change specializes Python byte storage. Host operations,
library exports, generated SDK interoperability and C byte storage remain
separate assignments.

The emitter recognizes underlying `uint8`, including `byte`, aliases, named
elements, named slices and arrays. Native `bytearray` backs byte slices and
arrays; other collections retain lists. Reads return ordinary Python `int`
values from 0 through 255. Existing Go integer normalization preserves byte
wrapping, and the slice store helper masks to eight bits. Array stores retain
the existing checked indexes and normalized integer operations.

Value-like headers hold backing identity, offset, length, capacity and a byte
hint. `BYTE_NIL` preserves native storage through nil reslices, empty spread
append and zero-argument append. Allocated empty slices have non-null empty
bytearrays. Runtime helpers return replacement headers without changing any
aliased header. Make and growth physically zero the entire capacity, including
spare elements. Growth retains `max(needed, max(1, 2*cap))`, allocates a distinct
backing and copies only the live elements. The existing make-size validation
and 2^32−1 host element limit remain. Native allocation rejects sizes above that limit
before allocation and reports unavailable memory as an implementation fault;
invalid make sizes and source bounds retain source runtime panics.

Fixed-length memoryview assignments prevent in-capacity append, copy and
array assignment from resizing existing backing. Byte append/copy use native
bytearray snapshots for overlap safety in both directions; they never create
a full-payload boxed list. Explicit append arguments construct a bytearray
from a tuple. String append/copy accept immutable Python `bytes` directly.
Clear zeros only the selected view. Nil, allocated-empty and empty offset
views accept clear/copy without reading nonexistent backing or changing nil
identity.

Array assignment updates existing backing, preserving array-view aliases;
array value copies and slice-to-array value conversions create independent
bytearrays, including zero-length arrays. Array equality compares native
values and array map keys snapshot immutable `bytes`. Named array and
interface identity continues to use the existing type descriptors. Go strings
already use Python `bytes`: conversions copy independently and preserve NUL,
high bytes and invalid UTF-8 without text decoding. Generic scalar consumers,
rune conversion and function-slice consumers continue to work with list
storage and Python integers. Arrays, headers, fields, pointers, interfaces,
closures and cooperative frame locals follow ordinary Python GC reachability.
Element addresses, slice-to-array pointer conversion and source generics
remain outside the accepted subset.

## Verification

Run from the Goalchemy root. `GOTOOLCHAIN` selects the compiler Go toolchain;
the loader and independent native-Go oracle keep their pinned defaults.

```sh
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec validate
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec generate -check
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./internal/emit/... ./internal/driver ./internal/subset ./internal/specgen -count=1
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=python go test ./tests/language -count=1 -v
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=go,typescript,python,java,csharp,rust,c FIXTURE=byte_storage go test ./tests/language -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -run TestPythonByte -count=1 -v
```

The catalog remains 13 type contracts, 94 function contracts, seven targets
and 502 generated files. Final checks passed without skips: the full contracts
suite in 89.352 s, the full Python language suite in 47.175 s, and the shared
byte fixture on all seven targets in 35.059 s. Driver and subset checks passed
in 17.342 s and 39.240 s respectively. A focused replay of accepted TypeScript,
Java and C# structural tests plus the three Python byte tests passed in
23.434 s. Final logs are `out/python-final-contracts.log`,
`out/python-final-language.log`, `out/python-byte-final-all-seven.log` and
`out/python-byte-final-focused.log`. The initial concurrent full contracts
run recorded the nil-clear host `TypeError` before the repaired runtime was
staged; that failure remains in `out/python-all-contracts.log`. The final
structural checks include nil/empty clear and string-copy regressions.
These durations are observations, not requirements.

`TestPythonByteStorage` imports the emitted common
fixture and checks actual one-byte `bytearray` storage, physical spare zeroing,
typed nil and allocated-empty values, header/backing identity, overlap in
both directions, detachment, arbitrary byte-string copies, bounds/make panic
categories, allocation faults before writes and generic list/rune behavior.
It invokes emitted byte array zero/clone/set/equality/key helpers, including
empty arrays and immutable snapshots, and checks emitted construction paths.
Matching output with boxed storage cannot pass these structural assertions.

`TestPythonByteGrowthNativeOracle` always compares against actual native Go.
The baseline printed `4 255 128 3 None`, then two `None` lines, recorded in
`out/python-byte-baseline.log`; the fixed result is `4 255 128 3 0`, then two
zero lines. The historical remaining-target baseline is retained. The earlier
opt-in regression now passes for Python too:

```sh
GOTOOLCHAIN=go1.25.14 GOALCHEMY_BYTE_TARGETS=python go test ./tests/contracts -run TestTypeScriptByteGrowthNativeOracle -count=1 -v
```

The shared `byte_storage` native-Go differential fixture covers aliases/named
types, nil/empty clear and copy, full slices, offset views, growth, overlap,
arrays and structs, assignment aliases, escaped pointers, interfaces,
closures, arbitrary strings and bounds. Default
`TestPythonByteValuesNativeOracle` reuses the target-neutral
`java_byte_values` source: array/interface map keys and snapshots, valid and
huge `uint64` indexes, invalid make sizes, closures and cooperative frame
storage across suspension. Full language regressions include non-byte
collections, runes, function slices, maps, interfaces and cooperative tasks.
Emitter/specgen packages currently have no package-local tests; these paths
are exercised by the contracts and language suites.

## Memory and throughput

Run each command in a fresh Python process:

```sh
python3 targets/python/tests/byte_storage_bench.py native 2
python3 targets/python/tests/byte_storage_bench.py boxed 2
python3 targets/python/tests/byte_storage_bench.py native 8
python3 targets/python/tests/byte_storage_bench.py boxed 8
```

Measured on Linux x86_64, kernel 6.8.0-87-generic, AMD Ryzen 7 6800H,
16 logical CPUs, CPython 3.10.12. The script warms the selected path, fills
source/destination buffers with the same 0–255 values, performs 16 copies,
and appends source to itself into doubled-capacity backing. All three
buffers remain reachable through payload assertions after requested GC.
Exact native element backing is four times the input size. The boxed
baseline uses the unchanged generic list path, including its native Python
list overlap temporaries. CPython caches these byte-valued integers, so this
baseline holds references to cached integers, rather than separately
allocating an integer for every element.

| Input | Storage | Exact native backing | Retained backing `getsizeof` | Copy 16 times | Append once |
| --- | --- | --- | --- | --- | --- |
| 2 MiB | bytearray | 8,388,608 B | 8,388,779 B | 8.405 ms | 3.187 ms |
| 2 MiB | Generic list/int | — | 67,811,048 B | 5,299.515 ms | 700.840 ms |
| 8 MiB | bytearray | 33,554,432 B | 33,554,603 B | 47.576 ms | 20.458 ms |
| 8 MiB | Generic list/int | — | 275,010,024 B | 13,843.415 ms | 1,680.666 ms |

The retained sizes sum `sys.getsizeof` for the three backing objects: native
bytearray data/header/trailing terminator or list reference slots and headers.
They exclude slice headers and shared cached integer objects. Exact payload
lengths establish one native byte per element; list over-allocation contributes
to the generic totals. The diagnostic also prints process peak RSS, which
includes temporary buffers and interpreter/allocator overhead. Timing and
RSS are illustrative observations, with no mandatory heap/RSS thresholds.
The output is `out/python-byte-memory.log`; timings were observed alongside
compiler checks and vary with process load. String conversions and overlap
snapshots still allocate native copies: this is an in-memory runtime.

## Remaining SDK requirements

Rust subsequently gained [native byte-storage evidence](rust-byte-storage.md).
C subsequently gained [native byte-storage evidence](c-byte-storage.md).
Existing generic non-byte append growth can leave unset spare capacity; this
change preserves that separate gap. Pending host operations, completion
retention, key/buffer lifetimes, cancellation, shutdown, library exports and
generated SDK packages remain Phase 4/5/6 work. GCE002 for unavailable
crypto/HTTP capabilities and GCE006 for unsupported library targets remain.

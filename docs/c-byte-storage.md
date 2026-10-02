# C byte storage evidence

This bounded Phase 4 component specializes C byte storage. Pending host operations,
library exports, generated SDK packages, actual TypeScript browsers, full pinned
Go SDK parity and real KAS interoperability remain separate requirements.

## Representation and semantics

The emitter recognizes all underlying `ir.U8` elements, including byte/uint8
aliases, named elements, named slices and named arrays. Arrays retain `GX_OBJ`
headers and slices retain `GX_SLICE` value headers. The existing `gx_V.pad` field
is an explicit native-byte hint (`1`) on these headers, including typed nil.
Payloads are actual one-byte `uint8_t` storage allocated with `GC_MALLOC_ATOMIC`;
other arrays/slices retain collector-scanned `gx_V` elements. `gx_vals` faults
if a native byte payload accidentally reaches a generic element consumer.

Headers preserve pointer, length, capacity and hint by value. Slice views retain
an interior pointer; full slices reduce capacity, while in-capacity append and
selected stores modify the original backing without resizing it. Growth detaches
using `max(required,max(1,2*capacity))`. Make and growth explicitly zero their
entire native capacity. Empty allocations reserve one collector-owned byte and
remain nonnil; nil and empty no-op appends preserve the complete original header.
Scalar reads produce `gx_int` values in 0..255, and stores wrap through `uint8_t`.
Signed and wide unsigned index/slice bounds are checked before pointer arithmetic
or writes. Make and growth validate source bounds and host limits before casts,
allocation or mutation; allocation also guards size multiplication.

Explicit append arguments use native `const uint8_t[]` construction. Byte spread
and string appends, copy, clear, slice-to-array conversion and native array
helpers avoid full-payload `gx_V` intermediates. Overlapping byte operations use
`memmove`. Empty paths return before null arithmetic or zero-size memory calls.
Clear touches only the selected live range. Go strings retain explicit binary
lengths: arbitrary NUL/high/invalid UTF-8 bytes copy independently in both
conversion directions without C-string truncation or a text decoder.

Generated array zero/clone/set/equality/key helpers execute native operations.
Assignment copies into the existing backing so slices retain their aliases;
value copies and slice-to-array value conversion detach, including zero arrays.
The key helper preserves the existing fixed-array encoding of scalar values;
encoded keys are snapshots and can use nine bytes per integer plus delimiters,
which differs from the one-byte element payload measured below. Struct and
interface descriptors preserve named type identity and key behavior. Fields,
escaped whole-array pointers, interfaces, closures and cooperative frame slots
copy the complete header. Unsupported element addresses, slice-to-array pointer
conversion and source-generics diagnostics are unchanged.

The remaining generic element consumers are element-specific rune conversion,
function-slice adaptation, task argument vectors, `lib.task.all`, tuples, structs
and map-key harness materialization. They operate on generic `gx_V` storage;
source map iteration returns individual keys. Contract decode, views and encode
support both backing representations; byte make contracts call the native
constructor. No shared compiler/linker scaffold, collector/thread setup, manual
freeing, pointer borrowing or export ownership rules changed.

## Collector and default tests

`TestCByteStorage` inspects emitted native construction and executes the actual
emitted zero, clone, in-place set, equality and key helpers. Actual `GC_size`
checks distinguish native payload allocations from generic `gx_V` arrays; a
`uint8_t` cast alone cannot establish native allocation. It then compiles the
real runtime for structural, binary, alias, overlap, zero-capacity, empty-path,
source-panic and fault checks. `TestCByteStorageSanitized` repeats that independent
runtime test with pinned LLVM 18.1.8 address/undefined-behavior sanitizers.
`ASAN_OPTIONS=detect_stack_use_after_return=0:detect_leaks=0` preserves the existing
Boehm stack-scanning requirement. Source panics must recover with the
`GX_RUNTIME_ERROR` descriptor; implementation faults must print the exact runtime
fault identity and exit with SIGABRT/status 134. Segfault/status 139, sanitizer
reports and ordinary process failures never satisfy either expectation.

The collector test executes nine modes in fresh processes, each with its own
distinct payload: escaped whole-array pointer, sole selected interior slice view,
struct field, interface box, closure environment, array value copy, direct native
slice, direct native array and cooperative frame storage. Obsolete allocation
stack slots are scrubbed before churn and eight explicit `GC_gcollect` calls;
every surviving byte is verified. The actual installed collector reports
`GC_get_all_interior_pointers()=1`. Ordinary runs observed 209–257 collections
per mode, and sanitizer runs observed 16–24. Existing collector initialization
and registered-thread behavior are retained.

Default C growth and values oracles compare against actual native Go. The shared
values fixture includes named arrays, array and struct/interface map keys,
independent snapshots, valid/huge uint64 indexes and make, closures and values
across `runtime.Gosched`. `TestByteValuesAcrossTargets` permanently replays the
strengthened struct-key fixture on all seven targets. The common source fixture
also covers named/alias slices, escaped pointers, fields and boxes.

The pre-change growth baseline is preserved in
`out/c-byte/historical-growth-baseline.log`: native Go expected
`4 255 128 3 0` followed by two zero lines; generic C printed `nil` in all three
spare positions. The specialized C default growth test passes. This repair does
not change the separate generic non-byte unset-spare-capacity follow-up.

## Reproduction and verification

Run from the Goalchemy repository root. Compiler/tests use Go 1.25.14; the loader
and independent native-Go oracle retain pinned Go 1.27.1. Tools here are GCC
11.4.0, pinned LLVM 18.1.8 plus libtinfo5, and BDWGC 8.2.8 with POSIX threads.

```sh
GOTOOLCHAIN=go1.25.14 GOALCHEMY_BYTE_TARGETS=c go test ./tests/contracts -run '^TestTypeScriptByteGrowthNativeOracle$' -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -run '^TestCByte' -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -run '^TestByteValuesAcrossTargets$' -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -count=1 -v
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=c go test ./tests/language -count=1 -v -parallel=4
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=go,typescript,python,java,csharp,rust,c FIXTURE=byte_storage go test ./tests/language -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./tests/sanitize -count=1 -v -parallel=4
GOTOOLCHAIN=go1.25.14 go test ./internal/emit/... ./internal/driver ./internal/subset ./internal/specgen -count=1
GOTOOLCHAIN=go1.25.14 go test ./tests/corpus ./tests/integration -count=1 -v -parallel=4
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec validate
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec generate -check
```

Final logs live under ignored `out/c-byte/final-*`; historical baseline and
initial scaffold logs remain labeled separately. Catalog/spec counts remain
13 type contracts, 94 function contracts, seven targets and 502 generated files.

All final required checks passed with no skips:

| Check | Result | Ignored log under `out/c-byte/` |
| --- | --- | --- |
| Strengthened default C tests | PASS, 20.027 s | `final-strengthened-default.log` |
| All-seven contracts and byte defaults | PASS, 272.586 s | `final-contracts-all-seven.log` |
| Full C language | PASS, 187.048 s | `final-c-language.log` |
| All-seven common byte fixture | PASS, 116.923 s | `final-common-all-seven.log` |
| Full C ASan/UBSan matrix | PASS, all 57 supported fixtures, 205.532 s | `final-c-sanitizers.log` |
| Emitters, driver, subset, specgen | PASS | `final-internal-checks.log` |
| All-target corpus and integration | PASS, 174.861 / 257.181 s | `final-corpus-integration.log` |
| Catalog and generated freshness | PASS, unchanged counts | `spec-validate.log`, `spec-freshness.log` |

The final test-only allocation assertions were replayed separately after the
broad checks; runtime files were unchanged. Shell syntax, local documentation
links and whitespace checks passed too.

## Fresh-process memory and throughput diagnostic

[Benchmark source](../targets/c/tests/byte_storage_bench.c) and
[wrapper](../targets/c/tests/byte_storage_bench.sh) build the actual C runtime.
The wrapper launches a new process for each native/generic and 2/8 MiB case.
It retains source, destination and doubled append output; verifies payloads,
capacities, collector ownership, input/output independence and one-byte versus
actual `sizeof(gx_V)` element storage. Generic mode uses current `gx_make_slice`,
`gx_copy` and `gx_append_slice` rather than a simulated estimate.

```sh
CC=cc CFLAGS=-O2 GOALCHEMY_BDWGC="$PWD/.toolchains/bdwgc" \
  sh targets/c/tests/byte_storage_bench.sh > out/c-byte/final-benchmark.log 2>&1
# Optional ignored build destination:
GOALCHEMY_BYTE_BENCH_OUT="$PWD/out/c-byte/benchmark-replay" \
  sh targets/c/tests/byte_storage_bench.sh
```

Each live backing capacity is N, N and 2N. With `sizeof(gx_V)=24`, exact native
retained element bytes are 4N, versus generic 96N. Collector allocation granules,
GC heap/free bytes and process peak RSS are measured separately. Final observed
results on Linux x86_64 under concurrent test/compiler load:

| N | Storage | Exact element bytes | Collector-rounded payload bytes | 16 copies (seconds) | Copy MiB/s | Append seconds |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| 2 MiB | uint8_t | 8,388,608 | 8,388,656 | 0.001229 | 26,029.40 | 0.002934 |
| 2 MiB | gx_V | 201,326,592 | 201,326,640 | 0.737159 | 43.41 | 0.151380 |
| 8 MiB | uint8_t | 33,554,432 | 33,554,480 | 0.007555 | 16,942.92 | 0.012611 |
| 8 MiB | gx_V | 805,306,368 | 805,306,416 | 2.649466 | 48.31 | 0.497061 |

Native/generic peak RSS was 10,240/313,728 KiB at 2 MiB and
34,816/1,577,472 KiB at 8 MiB. The corresponding GC heap sizes were
10,887,168/352,342,016 bytes and 34,025,472/1,694,527,488 bytes. These values
include allocator state and generic temporary snapshots. The generic 8 MiB case also prints Boehm's repeated-large-allocation
warning, retained in the log. Timings, heap and RSS are diagnostic observations,
with no acceptance thresholds. The benchmark wrapper's initial wrong GC-header
path failure is retained as `historical-benchmark-scaffold.log`; final execution
reports actual BDWGC 8.2.8 in each process. Byte backing is collector-owned with
no manual release; this remains an in-memory runtime rather than streaming.

## Remaining requirements

Generic non-byte append growth can still leave unset spare capacity. Host request
completion, cancellation/shutdown, buffer/key lifetimes, exported libraries and
SDK target packages remain Phase 4/5/6 work. Unavailable crypto/HTTP capability
and unsupported library-target diagnostics remain. These byte checks do not
establish generated SDK compatibility, actual browsers or real KAS acceptance.

# Rust byte storage evidence

This bounded Phase 4 assignment specializes Rust bytes. Pending host operations,
exported libraries, browser packaging and real-platform SDK interoperability
remain separate requirements. C subsequently gained [native byte-storage evidence](c-byte-storage.md).

## Representation and semantics

The emitter recognizes every underlying `ir.U8` element, including `byte`,
`uint8`, aliases, named elements, named slices and arrays. Arrays use
`V::Obj` handles to traced `Obj::Bytes(Vec<u8>)` leaves. Byte slices use
`V::ByteSlice(handle, offset, length, capacity)` over those same leaves.
Other storage retains `Obj::Vals(Vec<V>)` and `V::Slice`. `BYTE_NIL` retains the
byte hint through zero defaults, nil reslices, zero-argument append and empty
spread/string append; allocated empty slices have a live backing handle.
Existing integer values and type descriptors remain unchanged: reads produce
unsigned 0..255 `V::Int` values and writes retain source wrapping.

Headers copy by value. Full slices preserve the backing handle and reduce
capacity; in-capacity writes modify the selected range without resizing the
backing. Growth detaches using `max(required,max(1,2*capacity))` and allocates
physically zeroed native capacity before copying the live view and appended
values. Make zeroes the entire capacity too. Explicit append arguments use a
native byte array in emitted code. Byte spread snapshots use `Vec<u8>`;
overlapping copy within one backing uses `copy_within`, and distinct backings
use a byte snapshot and `copy_from_slice`. Clear uses `fill(0)` on the selected
view. Native paths never materialize full-payload `Vec<V>` intermediates.

Array zero/clone/set/equality/key helpers use native backing and byte snapshots.
Assignment copies into existing storage so array slices remain aliases; value
copies and slice-to-array value conversions detach, including zero-length
arrays. Keys contain immutable `Rc<[u8]>` snapshots under the existing named
and interface type identity. Structs, pointers, escaped views, interfaces,
closures and cooperative frames retain their source behavior. Go strings
remain `Rc<[u8]>`; string conversions copy arbitrary binary bytes independently
without a text decoder.

All byte index/store operations validate signed and unsigned-64 bounds before
accessing storage. Make/growth check lengths, capacities and host limits before
mutation or narrowing. Existing unsupported element-address, slice-to-array
pointer and generic diagnostics remain. Generic non-byte growth still has its
separate unset-spare-capacity follow-up.

The collector marks `ByteSlice` handles alongside `Obj`, `Ptr` and generic
slice handles, and treats `Obj::Bytes` as a leaf. Byte helper heap closures only
read or write native storage; they do not allocate heap objects or reenter the
`RefCell`. No new unsafe code or shared compiler/linker scaffolding was added.
Remaining `V::Slice`-only consumers include element-specific generic paths:
rune conversion, function-slice adaptation, `lib.task.all` and generic append
storage. `map_keys` is a contract-harness materialization with the current
`K=int` case; source map iteration returns keys individually rather than
constructing that slice. Contract decode, views and encode support both header
kinds.

## Independent verification

Run from the Goalchemy repository root. Build tools use Go 1.25.14; the loader
and actual native-Go oracle keep the pinned Go 1.27.1 default.

```sh
GOTOOLCHAIN=go1.25.14 go test ./tests/contracts -count=1 -v
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=rust go test ./tests/language -count=1 -v -parallel=4
GOTOOLCHAIN=go1.25.14 GOALCHEMY_TEST_TARGETS=go,typescript,python,java,csharp,rust,c FIXTURE=byte_storage go test ./tests/language -count=1 -v
GOTOOLCHAIN=go1.25.14 go test ./internal/emit/... ./internal/driver ./internal/subset ./internal/specgen -count=1
GOTOOLCHAIN=go1.25.14 go test ./tests/corpus ./tests/integration -count=1 -v -parallel=4
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec validate
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy spec generate -check
```

Default `TestRustByteGrowthNativeOracle` reproduces the original growth failure
against actual Go: Rust originally printed `4 255 128 3 nil`, then two `nil`
lines. The repaired result is `4 255 128 3 0`, then two zero lines. The original
failure is retained in `out/rust-bytes/baseline.log`; it is a historical
observation, not passing evidence. `TestRustByteValuesNativeOracle` reuses
`java_byte_values`: named arrays, array/interface map keys, immutable key
snapshots, valid/huge unsigned indexes, and cooperative suspension.

`TestRustByteStorage` inspects actual emitted native construction and compiles
and executes its native array zero/clone/in-place assignment/equality/key
helpers. The independent [runtime structural tests](../targets/rust/tests/byte_storage_test.rs)
check backing element size, all empty operations, nil/empty identity, alias
ranges, both overlap directions, zero spare capacity, binary string copies,
zero-length arrays, overflow before writes and generic consumers. Fault checks
require the implementation fault marker; source panics require a runtime-error
descriptor, so host bounds or borrow panics cannot satisfy them.

`TestRustByteGCNativeOracle` forces `GOALCHEMY_GC_THRESHOLD=1` through the common
escaped-pointer/interface/closure fixture and cooperative values fixture.
Structural collector coverage also roots distinct native allocations through
each of direct byte headers, array objects, pointers, fields, interface boxes,
closures and array copies. Each survives repeated safepoints independently.
No required checks are skipped or replaced with expected observations.

Final verification passed without skips: contracts and every seven-target
runtime harness plus all TypeScript/Java/C#/Python/Rust default byte tests
(89.449 s), all 54 Rust language fixtures (296.737 s), the common byte fixture
on all seven targets (106.567 s), seven all-target corpus regressions
(286.616 s), and integration examples/reproducibility/manifests/diagnostics
(370.312 s). Driver and subset checks passed in 123.468/212.475 s; emitter and
specgen packages have no separate test files. Catalog validation remains 13
types, 94 functions and seven targets; all 502 generated files are current.

Logs are `out/rust-bytes/contracts.log`, `language-rust.log`,
`all-seven-byte-storage.log`, `corpus-integration.log`, `compiler.log`,
`validate.log` and `freshness.log`. The initial `focused.log` and
`contracts-initial.log` retained a new inspection-test path error (`main.rs`
instead of `src/main.rs`); the corrected emitted helper execution passed in
`emitted-structural.log` and the full final contracts run. Compiler/toolchain
installation I/O overlapped the longer initial checks; no test was skipped or
restarted on those delays.

A generated Cargo release build also passed using only the standard library,
and its stdout/stderr matched the actual pinned Go 1.27.1 execution of the
common fixture byte-for-byte:

```sh
GOTOOLCHAIN=go1.25.14 go run ./cmd/goalchemy compile -target rust -out out/rust-bytes/cargo ./tests/language/testdata/byte_storage
cargo build --release --manifest-path out/rust-bytes/cargo/Cargo.toml
out/rust-bytes/cargo/target/release/main > out/rust-bytes/cargo.stdout 2> out/rust-bytes/cargo.stderr
GOTOOLCHAIN=go1.27.1 go run ./tests/language/testdata/byte_storage > out/rust-bytes/cargo-go.stdout 2> out/rust-bytes/cargo-go.stderr
cmp out/rust-bytes/cargo.stdout out/rust-bytes/cargo-go.stdout
cmp out/rust-bytes/cargo.stderr out/rust-bytes/cargo-go.stderr
```

Whitespace, shell syntax and local documentation link checks passed. No SDK,
reference production, lockfile or platform profile was edited.

## Fresh-process memory and throughput diagnostic

```sh
sh targets/rust/tests/byte_storage_bench.sh out/rust-bytes/bench > out/rust-bytes/benchmark.log 2>&1
```

The [benchmark source](../targets/rust/tests/byte_storage_bench.rs) compiles
against the current real runtime and runs four fresh processes: native/generic
storage at 2/8 MiB. Every process retains a source, destination and doubled
append result in registered roots, verifies every payload byte, and checks
actual backing lengths/capacities. It copies 16 times and appends once. Retained
element backing is exactly `4 * payload * element_size`, excluding Vec/object
headers, allocator metadata, root vectors, temporary snapshots and RSS effects.

Observed with Rust/Cargo 1.98.0, edition 2021, `opt-level=2`, `debuginfo=0`,
Rust's system allocator, default GC threshold 50,000, and one explicit final
collection. Each process retained three live traced objects (peak three).
The current generic `V` occupies 24 bytes per element; its scalar integers do
not require a separate allocation per byte.

| Input | Storage | Exact retained element bytes | RSS after collection | Copy 16 times | Copy MiB/s | Append once |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| 2 MiB | Vec<u8> | 8,388,608 | 12,525,568 B | 12.475 ms | 2,565.20 | 2.447 ms |
| 2 MiB | Vec<V> | 201,326,592 | 202,899,456 B | 697.549 ms | 45.87 | 127.938 ms |
| 8 MiB | Vec<u8> | 33,554,432 | 44,056,576 B | 46.551 ms | 2,749.65 | 9.794 ms |
| 8 MiB | Vec<V> | 805,306,368 | 806,703,104 B | 2,690.203 ms | 47.58 | 453.755 ms |

Native/generic maximum process RSS was 12,232/296,464 KiB at 2 MiB and
43,024/1,181,012 KiB at 8 MiB. RSS includes runtime/allocator state and temporary
snapshots; retained element bytes are measured separately and exactly. Timing
and RSS are diagnostic observations under concurrent compiler/toolchain load,
with no flaky acceptance thresholds. String and overlap snapshots still
allocate native copies; this runtime remains an in-memory implementation.

## Remaining SDK requirements

C subsequently gained [native byte-storage evidence](c-byte-storage.md). Generic
non-byte growth remains a separate follow-up. Host operations and completion
retention, key/buffer lifetimes, cancellation, shutdown, exported APIs and target
packages remain Phase 4/5/6 work. GCE002 for unavailable crypto/HTTP capabilities
and GCE006 for unsupported library targets remain. The full SDK objective still
requires all seven targets, actual TypeScript browsers, full pinned-Go-SDK
parity and real KAS interoperability.

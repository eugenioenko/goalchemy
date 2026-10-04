# Follow-ups

Known gaps and deferred improvements, recorded so they can be picked up later.

## C target

### Legacy scalar library boundary

Non-`main` packages build as C libraries (`goalchemy.h`, `libgoalchemy.a`).
The owned value API accepts booleans, integers, strings, floats and supported
struct/array/slice trees, including cooperative calls and error mapping.
The all-target [float consumers](float-support.md) exercise this API.
The older scalar entry points retain these limits:

- Parameters and results may only be integers, `bool`, `string`, `float32` and `float64`. Structs, slices and errors use the owned value API; maps and function values remain excluded from value exports.
- Exported functions must not suspend, so cooperative code cannot be exported.
- The host must call `goalchemy_init` and the exports from one thread. Other host threads are not registered with the collector.
- Returned strings point into collected memory, so the host must copy them before dropping its references.

See the [Go](go-library-boundary.md), [TypeScript](typescript-library-boundary.md), [Java](java-library-boundary.md) and [C#](csharp-library-boundary.md) boundaries for the other host representations.

## All targets

### Native byte storage and generic growth

TypeScript now uses `Uint8Array` for byte slices and arrays, including named
uint8 element types. See [design, verification and memory evidence](typescript-byte-storage.md).
Java now uses native `byte[]` for byte slices and arrays, including named
uint8 elements, with unsigned primitive reads and zeroed capacity. See
[Java design, verification and memory evidence](java-byte-storage.md).
C# now uses native `byte[]` for byte slices and arrays, including named
uint8 elements, with primitive `long` reads and zeroed capacity. See
[C# design, verification and memory evidence](csharp-byte-storage.md).
Python now uses native `bytearray` for byte slices and arrays, including named
uint8 elements, with immutable `bytes` string copies and zeroed capacity. See
[Python design, verification and memory evidence](python-byte-storage.md).
Rust now uses traced native `Vec<u8>` leaves for byte arrays and slices, with
unsigned `V::Int` reads, typed nil headers and zeroed capacity. See
[Rust design, verification and memory evidence](rust-byte-storage.md).
C now uses collector-owned native `uint8_t` arrays and slice backing, with
unsigned scalar reads, typed nil headers and zeroed capacity. See
[C design, verification and memory evidence](c-byte-storage.md).

The stronger growth-zeroing regressions recorded in these evidence documents
exposed unset spare capacity after append growth in generic storage.
The float-support follow-up now initializes new generic spare slots from the
declared element's zero factory, including nested arrays and structs. It
preserves existing backing contents when append reuses capacity, aliasing and
the established capacity rule. See [float verification](float-support.md).

### Host operations and portable libraries

Generated Go now provides pending host I/O and its cancellation/completion
lifecycle for the actual native HTTP transport. See [host-operation design and
bounded evidence](host-operations.md). TypeScript now has a generic portable
[Promise lifecycle and executable entry](typescript-host-operations.md), verified
in Node and an actual browser. Java now has a bounded [worker/future mailbox lifecycle](java-host-operations.md) and explicit monotonic executable entry. C# now has a bounded [Task/mailbox lifecycle](csharp-host-operations.md), monotonic executable entry and managed source-recursion guard. Python now has a bounded [Condition/Future mailbox lifecycle](python-host-operations.md), blocking monotonic executable entry and managed source-recursion handling. Rust now has a bounded [native mailbox/resource-ACK lifecycle](rust-host-operations.md), dedicated monotonic executable entry and generated recursion guards. C now has a bounded [native wire mailbox/resource-ACK lifecycle](c-host-operations.md), serialized monotonic executable entry and generated source guards.

Go, TypeScript, Java, C#, Python and Rust now provide production crypto/HTTP capabilities and importable value libraries. TypeScript's WebCrypto/fetch library graph works in Node and actual browsers; its executable wrapper remains Node-specific. Java uses JDK 21 native capabilities and pinned BC 1.86 for HKDF and omitted-public-point P-256 imports. C# uses .NET 8 built-in crypto and HttpClient with owned cancellable library calls; IEEE CRC32 uses the pinned first-party System.IO.Hashing NuGet package. Python uses maintained cryptography and bounded standard-library HTTP with owned cancellable sync/async calls; see the adjacent [Python TDF3 delivery](../../sdk/docs/generated-python-library.md). Rust provides maintained locked OpenSSL/reqwest capabilities and owned cancellable Cargo Result/Future libraries; see [its boundary](rust-library-boundary.md) and [Rust TDF3 delivery](../../sdk/docs/generated-rust-library.md). Production C adapters and its SDK library remain open, together with final all-target package/CI delivery checks in the adjacent SDK plan.

# Follow-ups

Known gaps and deferred improvements, recorded so they can be picked up later.

## C target

### Library mode is narrow

Non-`main` packages build as C libraries (`goalchemy.h`, `libgoalchemy.a`). Current limits:

- Parameters and results may only be integers, `bool`, and `string`. Structs, slices, maps, errors, and function values are rejected.
- Exported functions must not suspend, so cooperative code cannot be exported.
- C's scalar library boundary does not support the value/async exports available on Go, TypeScript and Java. C#, Python and Rust library builds still report GCE006.
- The host must call `goalchemy_init` and the exports from one thread. Other host threads are not registered with the collector.
- Returned strings point into collected memory, so the host must copy them before dropping its references.

Widening C's boundary remains necessary for SDK consumers: native value conversion, error mapping, owned outputs and async exports. See the accepted [Go](go-library-boundary.md), [TypeScript](typescript-library-boundary.md) and [Java](java-library-boundary.md) boundaries.

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
expose unset spare capacity after append growth in generic storage.
All six non-Go byte implementations now zero capacity; existing generic non-byte
append paths still leave spare capacity uninitialized. These follow-ups must
preserve aliasing and the established capacity rule.

### Host operations and portable libraries

Generated Go now provides pending host I/O and its cancellation/completion
lifecycle for the actual native HTTP transport. See [host-operation design and
bounded evidence](host-operations.md). TypeScript now has a generic portable
[Promise lifecycle and executable entry](typescript-host-operations.md), verified
in Node and an actual browser. Java now has a bounded [worker/future mailbox lifecycle](java-host-operations.md) and explicit monotonic executable entry. C# now has a bounded [Task/mailbox lifecycle](csharp-host-operations.md), monotonic executable entry and managed source-recursion guard. Python now has a bounded [Condition/Future mailbox lifecycle](python-host-operations.md), blocking monotonic executable entry and managed source-recursion handling. Rust now has a bounded [native mailbox/resource-ACK lifecycle](rust-host-operations.md), dedicated monotonic executable entry and generated recursion guards. C now has a bounded [native wire mailbox/resource-ACK lifecycle](c-host-operations.md), serialized monotonic executable entry and generated source guards.

Go, TypeScript and Java now provide production crypto/HTTP capabilities and importable value libraries. TypeScript's WebCrypto/fetch library graph works in Node and actual browsers; its executable wrapper remains Node-specific. Java uses JDK 21 native capabilities and pinned BC 1.86 for HKDF and omitted-public-point P-256 imports. Production C#/Python/Rust/C adapters and SDK library exports remain open, together with final all-target package/CI delivery checks in the adjacent SDK plan.

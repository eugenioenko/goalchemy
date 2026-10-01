# Follow-ups

Known gaps and deferred improvements, recorded so they can be picked up later.

## C target

### Sanitizer runs need a newer LLVM

`tests/sanitize` builds the C target with clang's address and undefined-behavior sanitizers. The machine's clang is 14 (the Ubuntu 22.04 default). Its AddressSanitizer predates support for high ASLR entropy, so on recent kernels (seen on 6.8) about one start in four segfaults before `main`, even for an empty program. The collector and generated code are not involved.

Current workaround: the `c-sanitize` runner (`internal/testutil/fixture.go`) runs programs under `setarch -R`, which disables address-space randomization for those processes only.

To fix: pin LLVM 18 or later, where the ASan fix landed. Either download the official release tarball into `.toolchains/`, as with the JDK and .NET, and have the sanitizer runner prefer it, or install a newer clang system-wide. Then drop `setarch -R`. This fits Milestone 11's pinned-toolchain work.

### Library mode is narrow

Non-`main` packages build as C libraries (`goalchemy.h`, `libgoalchemy.a`). Current limits:

- Parameters and results may only be integers, `bool`, and `string`. Structs, slices, maps, errors, and function values are rejected.
- Exported functions must not suspend, so cooperative code cannot be exported.
- Library builds are C-only; other targets report GCE006.
- The host must call `goalchemy_init` and the exports from one thread. Other host threads are not registered with the collector.
- Returned strings point into collected memory, so the host must copy them before dropping its references.

Widening this is the "library mode" work an SDK consumer such as OpenTDF needs: idiomatic host APIs per target, value conversion, error mapping, and async exports.

## All targets

### Byte slices are boxed

On every target except Go, a `[]byte` is a slice of boxed integers (one dynamic value per byte). That is correct but slow and memory-hungry for bulk data such as encryption and file I/O. Specialized byte-slice storage, and fast paths in the string conversions and `copy`, would fix it.

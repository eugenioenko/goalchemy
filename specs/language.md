# Goalchemy language specification

Version: 0.1 design baseline with accepted follow-ups. Updated October 4, 2026.

Goalchemy is a restricted Go-compatible source language for compilation into multiple target languages. Its source uses ordinary Go syntax and `.go` files. This document defines the intended source semantics and the boundaries of the initial implementation. A feature described here is not a claim of existing compiler support.

The implementation roadmap and runtime contract architecture are in [the project plan](../plan.md).

## 1. Authority and compatibility

Programs must parse and type-check under the selected Go source profile and pass Goalchemy's semantic whitelist. Successful Go type checking alone does not establish Goalchemy support.

The source language baseline is Go 1.25. The reference compiler is Go 1.27.1, configured with the selected packages' language versions capped at the Goalchemy baseline for source being transpiled. A package requiring a newer language version is rejected rather than silently interpreted using older rules.

The source platform is Linux amd64 with cgo disabled and explicitly configured build tags. The source meanings of `int` and `uint` are fixed at 64 bits on every output target. Cross-compilation does not adopt the target machine's native integer width or platform build tags.

For accepted constructs, Go 1.25 semantics apply except for the explicit implementation choices and cooperative execution contract below. This specification takes precedence over an emitter's host-language defaults. Canonical runtime contracts refine individual operations without contradicting it.

## 2. Feature gates

The first compiler release implements the sequential gate. The cooperative gate follows. A compiler release publishes its supported features and target capabilities, and rejects any operation outside them.

| Feature | Gate or disposition |
| --- | --- |
| Packages, imports, constants, variables, initialization | Sequential |
| Functions, methods, recursion, closures, variadic arguments, multiple results | Sequential |
| Named types and non-generic type aliases | Sequential |
| Boolean and fixed-width integer operations | Sequential |
| `float32`, `float64`, numeric conversions and floating-point operations | Sequential |
| Strings, structs, fixed arrays, slices, maps | Sequential |
| Struct embedding and promoted fields/methods | Sequential |
| Pointers to locals, allocated values, and struct fields | Sequential |
| Basic interfaces, assertions, and type switches | Sequential |
| `if`, expression switch, `fallthrough`, ordinary `for`, supported range forms | Sequential |
| Return, break, continue, labeled break and continue | Sequential |
| `defer`, `panic`, `recover`, errors as values | Sequential |
| Tasks, channels, channel range, `select`, synchronization, cancellation | Cooperative |
| Generic declarations, generic aliases, generic methods, instantiations | Rejected in 0.1 |
| Range over iterator functions | Rejected in 0.1 |
| Complex values or operations | Rejected |
| Pointers to slice or array elements | Rejected in 0.1 |
| `unsafe`, source reflection, cgo, `go:linkname`, `goto` | Rejected |
| Finalizers, source-visible destructors, explicit parallel execution | Excluded |

Supported sequential range operands are arrays, pointers to arrays where no prohibited element address is formed, slices, strings, maps, and integers. Channel range requires the cooperative gate.

Compiler specialization of runtime operations is an implementation technique and is permitted without source generics. An imported generic call is still a generic source instantiation and is rejected in 0.1 unless replaced by an explicit non-generic source override.

## 3. Names and program structure

Go rules govern scope, declarations, identifiers, method sets, embedding, and type identity. Defined types preserve their identity through lowering. An alias refers to its aliased type.

The build configuration selects executable or exported entry points. Reachability includes required type information, conservative interface dispatch candidates, and package initialization. Imported executable behavior must be included as accepted source or mapped to an explicit capability.

Package initialization occurs once per program instance, in Go dependency order. Within a package, the compiler uses stable source-file ordering and preserves Go's variable dependency ordering and `init` behavior. Mapped packages with initialization effects require an explicit initialization contract. Initialization cannot silently be discarded.

Compiler directives have no implicit authorization. Build constraints are handled by the Go loader; any directive that changes executable semantics requires explicit support. Unsupported compiler directives produce diagnostics.

## 4. Primitive values

### Booleans and integers

Boolean expressions follow Go's typing and short-circuit rules.

Integer types include `int8` through `int64`, `uint8` through `uint64`, `int`, `uint`, and their normal `byte` and `rune` aliases. `uintptr` is excluded from the initial profile. Compile-time integer constants use arbitrary precision and Go's representability rules. Typed runtime operations use their specified widths.

Arithmetic, bitwise operations, conversions, shifts, division, remainder, and overflow follow Go's rules. Signed operations must not inherit undefined behavior from C, and JavaScript numeric rounding must not affect integer results. Division by zero and invalid runtime shift counts produce source panics as specified by the corresponding runtime contracts.

Lengths, capacities, and indices use the 64-bit source profile. A backend may reject an allocation because host resources are insufficient; it may not silently truncate a valid integer or allocation size. Resource exhaustion is a runtime failure, not a successful alternate semantic result.

### Floating-point values

`float32` and `float64` represent IEEE 754 binary32 and binary64 values on all
seven targets. Named types, aliases, decimal and hexadecimal literals, and
typed or untyped constants follow Go's type and representability rules.
Constants retain arbitrary precision until they are rounded directly to the
destination width.

Arithmetic rounds each operation to its declared width using nearest,
ties-to-even rounding. Explicit conversions preserve their rounding effect,
including conversions to the same type; Goalchemy selects unfused execution.
Subnormal values, signed zero, infinities and NaN comparisons retain their IEEE
behavior. Runtime division by zero uses the selected native-Go IEEE behavior.
NaN payload bits are unspecified. `min` and `max` propagate NaN and select
negative and positive zero respectively when comparing signed zeros.

Representable float-to-integer conversions truncate toward zero. Integer-to-float
conversions round directly to the destination precision, without first rounding
through another float width. For unrepresentable runtime float-to-integer
conversions, Go leaves the result implementation-dependent; Goalchemy selects
the pinned Go 1.27.1 Linux-amd64 results as defined by
[`core.float.convert`](runtime/core/float_convert.yaml).

Floats work in the supported structs, arrays, slices, maps, interfaces,
cooperative frames and exported value boundaries. Zero initialization produces
positive zero. Map keys compare positive and negative zero as equal; keys
containing NaN never match an existing entry, including aggregate and interface
keys. Native host map equality must not change those rules.

Builtin `print` and `println` use the pinned reference toolchain's shortest,
width-aware float spelling. This feature does not add complex numbers or the
full Go `math` library. See [float support](../docs/float-support.md) for the
verification scope and native boundary representations.

### Strings

A string is an immutable sequence of arbitrary bytes. It is not required to contain valid UTF-8. Its length and indices are byte-based. Iteration decodes UTF-8 with Go's invalid-encoding behavior and produces byte positions and runes.

Concatenation and slicing preserve bytes. Conversions between strings, byte slices, and rune slices preserve the Go-defined values and copy/share behavior. A host Unicode string is only an adapter representation after an explicit conversion.

## 5. Aggregate values and identity

### Structs and arrays

Assignment, argument passing, and return of structs and arrays copy their values. Copying proceeds through value fields; reference-bearing fields retain their referenced identity. A copied struct containing a slice therefore has an independent slice header sharing the original backing storage.

Value receiver calls receive a value copy. Pointer receiver calls refer to the original storage. Promoted member access follows the path resolved by `go/types`, including required nil checks and receiver adaptation.

### Slices

A slice consists semantically of a backing-storage reference, offset, length, and capacity. A nil slice has no backing storage and has zero length and capacity. An empty non-nil slice remains distinguishable from nil.

Assigning a slice copies its header. Existing aliases observe element mutations through their shared storage. Indexing checks length, not capacity. Slicing applies Go's bounds rules and preserves backing identity. Full slice expressions constrain capacity.

Append reuses storage when capacity suffices. When growth is required, it allocates a new backing array, copies existing elements, appends the new values, and returns a new header. Existing aliases retain their old backing storage. Overlap and element-copy semantics must match the operation contract.

Goalchemy fixes its growth capacity at `max(required_length, max(1, 2 * old_capacity))`. Growth arithmetic is checked before allocation; capacity is never allowed to wrap. If doubling exceeds the source capacity limit while the required length remains representable, use the maximum representable capacity. Actual allocation can still fail from resource exhaustion.

`make` uses the explicitly requested capacity, or length when capacity is omitted. Fresh byte/rune slices created by string conversion use capacity equal to length. These choices are consistent across Goalchemy targets; native Go need not choose identical incidental capacities.

### Maps

A map value refers to shared mutable map storage. Copying a map value preserves that identity. A nil map has length zero, yields a missing lookup with the element zero value, and permits deletion and clearing; insertion into it panics.

Key equality follows the accepted Go key type. Structs and arrays can be comparable keys. Interface keys require a comparable dynamic value or the operation panics. A host dictionary's key equality is not automatically sufficient.

The source cannot require a particular map iteration order. Goalchemy's baseline runtime traverses entries in insertion order, snapshots entry identities when iteration starts, skips entries deleted before visitation, and excludes newly inserted entries. Updating an existing entry preserves its position and its current value is read when visited. Deletion and reinsertion creates a new entry identity. Native-Go comparisons allow all Go-permitted iteration outcomes.

### Pointers and storage

Pointers identify logical storage locations. A location includes object identity and a field path where needed, so a pointer to a field remains valid when the enclosing value is assigned. Copying a pointer preserves identity; copying the pointed-to struct or array follows value semantics.

Supported addressable forms include locals, globals, allocated values, and struct fields. Slice and array element addresses are rejected initially. Pointer arithmetic and conversions between pointers and integers are rejected.

Nil dereference panics. Targets must preserve permitted pointer equality observations without exposing host addresses. Identity-sensitive comparisons involving distinct zero-sized allocations follow Go's permitted outcomes and are not used as exact native-oracle fixtures.

## 6. Interfaces and errors

An interface value contains a dynamic type and dynamic value, or is a nil interface. An interface containing a typed nil pointer is not a nil interface. Boxing a value type copies that value according to its normal semantics.

Method dispatch, type assertions, comma-ok assertions, and type switches use preserved Go type identity and method sets. Interface equality follows Go's rules, including panics for non-comparable dynamic values where required.

`error` remains an interface satisfied by source-defined types. Errors cannot be reduced universally to a message and one type tag. Registered error capabilities preserve identity and support the specified `Is`, `As`, wrapping, and unwrapping behavior, including custom source methods and multiple wrapped causes when those operations are enabled.

Returned errors remain ordinary return values. Generated code preserves messages and fields observable by source code. Adapter conformance may normalize native-library error messages only when the adapter contract explicitly permits it.

## 7. Calls and control flow

Calls preserve receiver adaptation, argument values, evaluation constraints, variadic construction, result arity, and copying. Calling a nil function value panics.

Closures capture storage according to the Go source profile. The compiler preserves per-iteration bindings for loop variables declared by the loop and does not accidentally turn distinct captures into one shared variable.

Assignments evaluate their operands and destinations according to Go's sequencing rules before applying stores. Multiple assignment cannot be implemented as unrelated sequential assignments. Short-circuit expressions evaluate only the required operands.

Where Go permits several operand evaluation orders, Goalchemy selects left-to-right evaluation while respecting stronger Go sequencing constraints. Differential fixtures distinguish this selected order from behavior Go does not specify.

Switches preserve case evaluation and control flow, including explicit `fallthrough`. Loop and label lowering preserves the destination of each break or continue. Arbitrary jumps through `goto` are excluded.

## 8. Deferred calls and panic

Executing `defer` evaluates and captures the deferred callee and arguments at that point. Each function invocation owns its own defer stack. Deferred calls execute in reverse registration order when the invocation returns or unwinds through panic.

Named result variables remain addressable during deferred execution. Deferred functions may change the eventual results. Deferred calls inside a loop accumulate until the containing function exits.

Panic carries a source value and unwinds the current task. `recover` obeys Go's direct deferred-call restrictions and returns the appropriate source panic value. Nil panic arguments follow the reference language behavior and must not be mistaken for an absence of panic. Nested panics and deferred calls must preserve Go's control rules.

A backend may implement unwinding through host exceptions or explicit status values, but must distinguish source panic from compiler/runtime implementation faults. Expected runtime panic categories are specified independently of host exception messages.

## 9. Cooperative execution

The cooperative gate extends the sequential language with tasks, channels, select, synchronization, and context operations. It does not introduce simultaneous source execution.

Each program instance owns a scheduler. Runnable tasks execute FIFO, and a running task keeps control until it waits, explicitly yields through a declared capability, returns, or panics. Runtime calls declare whether they may suspend; ordinary arithmetic, copying, and allocation do not introduce scheduling points.

Spawning a task evaluates its callee and arguments immediately in the parent and then enqueues the child. Host I/O can finish independently, but completions resume source code only through the scheduler. Host callbacks enter through the same scheduler and do not reenter a suspended source stack arbitrarily.

Channels preserve rendezvous, buffering, close, zero-value receive, comma-ok, and nil-blocking behavior. Sending to a closed channel and closing a closed or nil channel panic. A channel receive drains queued values before reporting closure.

Select registers all eligible waits and commits exactly one case. If several cases are ready, the scheduler's choice source selects uniformly among them. Tests inject the choice source. The losing waits are removed before continuation. A default case runs only when no communication can proceed.

Mutexes and WaitGroups have real state and waiting behavior despite single-task execution. Cancellation is an explicit event observed by operations that consult the associated context. A deadline uses the configured clock. Cancelling a context does not forcibly unwind unrelated source instructions.

Returning from an ordinary function does not implicitly cancel its child tasks. Returning from executable `main` terminates the program instance. An embedded program instance retains background tasks until completion or explicit host shutdown. Shutdown behavior is part of the host boundary contract.

If no task can run and no external event or timer can make progress, the runtime reports a blocked/deadlocked instance to the host or harness. Expected blocked fixtures are distinguished from unexpected deadlock. Preemptive busy-wait synchronization is outside this execution model.

## 10. External boundaries and lifetime

An external call must resolve to a declared capability symbol with a canonical contract and a compatible target implementation. Dependencies on a native library are contained within that implementation.

Capabilities use supported types, declared records and interfaces, or explicit opaque handles. They specify callback effects, ownership or retention of inputs, mutations, returned errors, panics, and cancellation behavior. An implementation cannot retain a borrowed buffer beyond its declared lifetime.

Programs use explicit close/release operations for resources whose lifetime is observable. Garbage collection is unobservable to source programs and does not replace resource cleanup. Host handles retaining source objects must register roots until released.

Cycles are permitted in the semantic object graph. Each target must preserve them and provide its selected collection strategy. Weak references cannot be substituted silently for strong source references.

## 11. Diagnostics and support claims

A program imports only its own module's packages and Goalchemy library packages under `github.com/eugenioenko/goalchemy/lib/`; every other import, including the Go standard library, is rejected. Library packages keep standard names (`github.com/eugenioenko/goalchemy/lib/sync`, `github.com/eugenioenko/goalchemy/lib/errors`, `github.com/eugenioenko/goalchemy/lib/context`, `github.com/eugenioenko/goalchemy/lib/time`, `github.com/eugenioenko/goalchemy/lib/runtime`) and are native Go wrappers when the program runs with the Go toolchain.

Unsupported syntax, types, operations, imports, effects, and target capabilities are compile errors. Each diagnostic includes a stable code, source span, symbol context when available, and a concrete remedy.

The compiler reports independently discoverable errors together. It may stop analysis of a malformed expression or package when Go typing cannot provide enough information, and must identify that limitation.

A target is supported for a feature only after its runtime and emitted code pass the corresponding conformance tests. Missing target support is reported before runnable output is claimed. Silent fallback to host semantics is prohibited.

## 12. Conformance and evolution

Tests compare return values, mutations, aliases, identities where defined, emitted bytes, errors, panics, and relevant synchronization outcomes. Stdout alone is insufficient.

Native Go is the oracle for specified Go behavior. Goalchemy's canonical contracts define its selected implementation-dependent choices. All Goalchemy targets must agree on those choices under controlled inputs and scheduling. Resource exhaustion, native addresses, and uncontrolled wall-clock timing are not exact equality observations.

Adding a language feature requires a specification update, acceptance and rejection rules, IR and lowering support, runtime contracts, and conformance cases. Source generics can later be introduced through concrete specialization without requiring generic syntax in C output, but they remain rejected in version 0.1.

Implementation may evolve incrementally. A semantic change is explicit, versioned, and accompanied by tests; it is never inferred merely because one backend accepts a construct.

## References

- [Go specification](https://go.dev/ref/spec)
- [Go defer, panic, and recover](https://go.dev/blog/defer-panic-and-recover)
- [Go type information](https://pkg.go.dev/go/types)

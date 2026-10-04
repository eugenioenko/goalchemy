# Float support

This work adds `float32` and `float64` throughout Goalchemy's seven targets:
Go, TypeScript, Python, Java, C#, Rust and C. Both precisions are covered by
all-target differential fixtures and native exported-library consumers.
Complex numbers and the full Go `math` library are outside scope.

## Language and runtime requirements

The source types represent IEEE 754 binary32 and binary64 values. Named types,
aliases and untyped floating-point constants follow Go typing and constant
representability rules. Decimal and hexadecimal literals are included. Constant
expressions must retain Go constant precision until conversion to a runtime
type; emitting a constant via an intermediate binary64 value must not introduce
double rounding when the destination is binary32.

Runtime operations include unary plus and minus, addition, subtraction,
multiplication, division, ordered comparisons, equality, inequality, increment,
decrement, compound assignments and the existing `min`/`max` builtins. Preserve
subnormal values, negative zero, infinities and unordered NaN comparisons. NaN
payload bits are not a portable source guarantee. `min` and `max` must propagate
NaN and select the correct sign of zero, independent of argument order.

Use a consistently rounded, unfused execution profile. In particular, explicit
floating-point conversions retain their rounding effect, including conversions
to the same type. Floating-point runtime division by zero follows the native
Go/Linux-amd64 IEEE behavior rather than Python's exception behavior.

Support conversions among both float widths and all supported signed and
unsigned integer widths. Representable float-to-integer conversions truncate
toward zero. Integer-to-float conversion must round the source integer directly
to the destination width; a path through binary64 can misround a binary32
result near a midpoint. Runtime conversions whose results cannot be represented
must have an explicit consistent policy and must succeed as required by Go,
rather than depend accidentally on host exceptions, undefined C casts or Rust
saturating casts. Goalchemy selects the results of the pinned Go 1.27.1
Linux-amd64 reference for implementation-dependent conversions: small integer
widths and signed `int32` use a signed-32 conversion followed by wrapping;
`uint32`, `int64` and `uint64` use the signed-64 profile described in
[`core.float.convert`](../specs/runtime/core/float_convert.yaml). NaN, infinity
and out-of-range values therefore produce consistent results on every target.
This policy is a deliberate platform choice, not a portable Go guarantee for
unrepresentable runtime conversions.

These rules derive from the [Go language specification](https://go.dev/ref/spec),
especially [numeric conversions](https://go.dev/ref/spec#Conversions),
[floating-point operators](https://go.dev/ref/spec#Floating-point_operators)
and [min/max](https://go.dev/ref/spec#Min_and_max). Go permits unfused operations
and specifies implementation-dependent results for unrepresentable runtime
numeric conversions; those cases must be distinguished from defined-result
differential tests.

## Values, collections and public APIs

Float values must work in every existing supported position: variables,
parameters, multiple results, closures, deferred calls, cooperative frames,
named types, structs, arrays, slices, maps and interfaces. Preserve existing
value-copy and slice-aliasing rules. Zero initialization must produce positive
zero, including allocated spare slice capacity.

Map equality must retain IEEE equality. Positive and negative zero address the
same key, while a NaN key never equals itself. Repeated NaN insertions create
distinct entries; lookup and deletion with a NaN do not find an earlier entry.
The same behavior applies to comparable structs, arrays and interface keys
containing float values. Host map equality is not sufficient by itself.

Each target's existing exported value boundary must admit scalar floats and
floats nested in its supported structs, arrays and slices. Preserve named type
identity internally, native result ownership, malformed-input validation and
the existing synchronous/asynchronous API behavior. Scalar values may use
native float representations; choosing optimized float array storage is not
required for semantic support.

Public Go, Java, C#, and Rust APIs use native `float32`/`float64`, `float`/`double`
or `f32`/`f64` types. TypeScript uses `number`, with explicit binary32 rounding;
Python uses `float`, accepting numeric inputs while rejecting booleans and
malformed values. C's owned value boundary uses `GXC_FLOAT` with a `double`
payload, rounding binary32 input explicitly. Existing collection ownership and
nil/empty conventions remain in effect.

## Acceptance

Acceptance requires native-Go differential fixtures for all seven targets,
meaningful native exported-library consumers on all seven targets, rejection
tests retaining the exclusions for complex numbers and `uintptr`, canonical
runtime contract coverage, generated-spec checks and passing hosted CI.

The differential fixtures must exercise ordinary arithmetic and exceptional
IEEE values, both precisions, direct integer conversion at rounding boundaries,
typed and untyped constants, explicit rounding, comparison/switch behavior,
float map and aggregate keys, collection copying/aliasing, interface equality,
and cooperative storage where supported. Direct `print`/`println` cases must
verify both float widths and exceptional values against the pinned Go 1.27.1
reference; its builtin float printing uses the shortest round-trip spelling,
unlike older Go releases. Native consumers must exercise both
scalar and nested exported values, result ownership and invalid inputs where
the boundary validates dynamic values. Assertions should observe exact defined
results and semantic properties rather than rely on host decimal formatting
or a particular NaN payload.

The pull request's hosted checks provide the current CI acceptance status.

Hosted CI explicitly runs the float language fixtures on all seven targets.
Its normal short suite otherwise runs language fixtures only on Go and
TypeScript; all-seven runtime contracts alone cannot verify compiler-generated
map equality or exported collection boundaries.

Float contracts may use the existing `native_tests` mechanism rather than add
a second float encoding to the canonical JSON harness protocol. Such contracts
must name tests that actually execute every declared operation on all seven
targets. Contract validation alone checks that the named files exist; the
explicit language and native-consumer CI tests provide the execution evidence.
Existing codec-backed runtime contracts still require all-seven conformance
and generated-spec freshness.

## Verification

IR tests cover direct rational-to-binary32 constant rounding and an
integer-valued float constant verifier regression. The emitted `floats` and
`co_floats` fixtures compare both precisions against native Go on every target,
including aliases, default constants, closures, deferred captures, variadic
results, direct integer rounding, explicit conversion barriers, overflow and
underflow, NaN map keys, aggregate/interface equality, append-growth zeros,
source channels and width-aware printing. These fixtures have passed locally.

An independent native-Go conversion probe confirmed that
`int64(4611686293305294849)` rounds directly to binary32 bits `5e800001`,
while conversion through binary64 produces `5e800000`. The unsigned case
`uint64(9223372586610589697)` likewise distinguishes `5f000001` from
`5f000000`. The differential fixtures cover these direct-conversion boundaries.

Local probe output and logs remain in ignored `out/` storage.

Independent checks of all six non-Go numeric helpers passed
16,096 float-to-integer comparisons and 8,048 integer-to-float bit comparisons
per target against Go 1.27.1/Linux-amd64. The fixed-seed probes include directed
NaN, infinity, zero, integer-range and rounding boundaries as well as random
inputs. These are helper checks; the emitted programs and native library
boundaries require their own acceptance evidence.

Native exported-library consumers are written for all seven targets in
`tests/integration/float_library_test.go` and its fixture directory. They cover
both IEEE scalar widths, nested float values, input/result ownership, source
channel suspension, global-result ownership and malformed inputs where the
existing host boundary admits them. Mixed-width multiple results are also
checked. All seven consumers have passed locally. Java uses boxed scalar
result types as required by its generic operation API; TypeScript and Python
validate scalar floats recursively in nested inputs. These consumers run in
the CI short suite without target skips.

An independent printing audit compares 8,014 raw IEEE values per non-Go host
against builtin printing in the pinned native Go toolchain. All six non-Go
hosts have passed. The audit exposed ECMAScript's different
decimal tie-breaking; TypeScript now rounds exact IEEE rational values using
ties-to-even before choosing the shortest spelling. The directed case is
retained in the language fixtures.

The compiler/IR/subset/contract/emitter unit checks and all-seven canonical
conformance have passed locally. The CI workflow also runs `go vet ./...`,
`go run ./cmd/goalchemy spec generate -check`,
`go test -short -timeout 30m ./...`, the all-seven float fixtures and
`TestTargetConformance` without the short flag. The pinned Python and native
build dependencies must be prepared with `scripts/ci-bootstrap.sh` first.

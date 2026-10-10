# Library handles

Exported functions and methods can return and accept pointers to exported
struct types of the root package. The host receives an opaque handle with the
type's exported methods. Supported targets: Go, TypeScript and Python. Other
targets reject handle exports with `GCE007` until their bindings land (#22).

```go
package counter

type Counter struct{ n int }

func New() *Counter                      { return &Counter{} }
func (c *Counter) Add(d int) int         { c.n += d; return c.n }
func (c *Counter) Close() error          { return nil } // optional
```

| Target | Handle | Release |
|---|---|---|
| Go | `*counter.Counter` with methods taking `ctx` and returning `error` | `Close() error` |
| TypeScript | `class Counter` with Promise-returning methods | `close()`, `await using` (`Symbol.asyncDispose`) |
| Python | `class Counter` with methods returning a library operation | `close()`, `with` and `async with` |

## What crosses

- A handle type is an exported root struct `T` whose pointer `*T` appears
  directly in an export signature, or in the signature of an exposed method of
  another handle.
- Exported methods of `T` and `*T` are exposed when every parameter and result
  is a copied value, a handle, a leading `context.Context` or a final `error`.
  Other methods are not exposed. A method named `Close` must be `Close() error`.
- Handles are direct parameters and results only, not fields of structs or
  elements of slices. A nil pointer crosses as the host's null.

## Instances

The first call that returns a handle without receiving one creates an
instance. The instance owns its source globals, runs package initialization
once, and keeps native resources such as crypto keys until it retires. Handles
created by later calls on the instance (`c.Child()`) join it. Free functions
without handle arguments still run in a fresh owner with fresh globals.

Handles from different instances cannot be mixed in one call: that fails with
`instance_mismatch`. Returning the same pointer twice returns the same host
object while that object is alive.

## Concurrency

Calls are serialized across the whole library, including calls on different
instances: each call loads its instance's globals, runs, and saves them. Calls
from several host threads are safe. Source `sync.Mutex` keeps Go semantics
inside a call.

Goroutines still running when a handle call returns are abandoned with a
warning. A host callback that calls back into the library deadlocks, as for
free functions.

## Close and failures

- `Close` fails calls queued on the handle with `closed`, cancels its active
  call (kind `canceled`), runs the source `Close` if there is one, then releases
  the handle. Later calls fail with `closed`; `Close` is idempotent. A source
  `Close` error is returned after the handle is released.
- The instance retires when its last handle is released: its globals and native
  resources are dropped.
- A handle collected without `Close` is released by a finalizer, which never
  runs source code and logs a warning.
- A source panic, fatal error or host fault during an instance call poisons the
  instance: later calls fail with `poisoned`, and `Close` only releases.

Warnings go to standard error. Go hosts can replace `rt.LibraryWarn`,
TypeScript hosts `setLibraryWarn` in `rt/runtime/library.js`, and Python hosts
`rt.set_library_warn`.

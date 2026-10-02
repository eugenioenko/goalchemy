# C pending host operations

This bounded Phase 4 port implements generic C native-wire mailbox operations,
separate resource-cleanup ACK and an explicit monotonic entry for cooperative
executables. Production HTTP/crypto/encoding adapters and SDK libraries remain
gated. It changes no unsupported `lib.*` mapping or GCE006 gate. Importable
seven-target TDF3 packages and real OpenTDF/KAS delivery remain required.

## Executable entry and ownership

Compiler-emitted cooperative `main.c` exposes `goalchemy_run_host()`, which calls
`gx_run_main_host` with the actual zero-global initializer and entry-frame factory.
It returns 0 after main and native cleanup, 1 when another executable entry owns
the process-static ABI, and 3 for a recognized host implementation fault. Source
panic/fatal reports terminate the executable with exit 2 **after** retirement.
`gx_host_main` is the terminating executable helper. These functions are not
SDK exports or a library ABI. Default `main`, `sh run.sh`, harness and sequential
execution retain deterministic virtual time and legacy behavior.

The host runs source on a dedicated 8 MiB collector-registered owner thread.
`GC_pthread_create`/`GC_pthread_join` balance its collector lifecycle. The existing
legacy sequential/virtual executable uses its collector-registered 1 GiB thread.
A common atomic reservation precedes source globals, scheduler and panic binding
changes. Overlapping host entry returns 1; overlapping terminating executable,
harness or reset entry is a protocol fault and aborts before changing the active
owner. Construction/submission failures represented by the runtime's fault
boundary release the reservation after cleanup. Failure to join a successfully
created owner cannot prove it has stopped: it aborts while keeping the reservation.

The source ABI still has process-static globals, `gx_sched`, `gx_handler` and
`gx_thrown`. Every emitted entry zeroes globals, then runs source initialization
again. Actual two-call output is `init 1`, `main 2` twice. This is neither a
package-init guard nor initialize-once library semantics, persistent clients,
independent instance globals or supported overlapping source execution. Later
minimal façades must meet the [SDK library requirements](../../sdk/docs/library-requirements.md).

Only the owner mutates source tasks/frames/results, contexts, channels and sync
objects. Supported captured context cancel, direct/select waiter completion and
waiter detach callbacks check owner/thread before touching fields and become
no-ops for native or retired owners. Active queue attach/dequeue/try-receive paths
also check first. Direct source primitive use from a native thread is a protocol
violation and aborts; it cannot longjmp into the owner's stack. The public header
is an internal runtime contract, not hostile-pointer isolation or an SDK boundary.

## Wire records, roots and resource cleanup

`gx_host_register` installs a scanned pending record, decoder roots and task park
**before** submission. The returned `gx_HostToken` contains only a malloc mailbox
pointer and numeric generation/operation/task identity. Token copies require
explicit retain/release; the owner and retained tokens share mailbox references.
Retiring closes/discards its bounded live table; a legal retained stale token can
publish/ACK safely afterward, and the last release destroys the mailbox. Tokens
and mailbox records never contain source frames, contexts, descriptors or inputs.

`gx_host_publish` snapshots arbitrary unsigned bytes or an explicit fault string
into native malloc ownership. Workers receive malloc-owned plain input snapshots
and IDs, publish records, and wake the owner. C cannot catch arbitrary native
exceptions/crashes: workers must translate unexpected implementation failures
into fault records and release acquisitions on every supported path. Declared
transport failures are ordinary wire data, decoded into source errors by the
owner. Decoders return an explicit fault string; unexpected source panic or
runtime nonlocal escape inside decode is converted to a host fault. Registration
cleanup still runs exactly once when cancellation-result decode faults.

Exact live generation/operation/**task** validation precedes both publication and
ACK. Wrong-task faults/results/ACKs cannot poison canonical work or release its
queued result. First applicable publication applies once after its own ACK;
publication FIFO survives removal of ACK-ready holes before later insertions.
Duplicate, foreign, stale and completed identities consume no valid work. Live
records, ACK flags, source pending roots and completed task lists prune promptly.

Resource ACK means native work has released transport/process/body/key/input
acquisitions. A cancellation flag, closed socket, dropped token or finished
native handle alone is insufficient. Adapter registration holders may retain
coordination metadata until owner cleanup; that cleanup joins workers and frees
holders before source resumption/entry return. A broken adapter that never ACKs
prevents safe retirement; the runtime invents no release. Native allocation or
publication failure must be handled by the adapter rather than silently dropping
its sole terminal signal.

Cancellation observed before ordinary result application wins; committed success
stays final. An applicable adapter fault beats cancellation and bypasses source
recover. Recognized owner-side `gx_fault` checks (such as allocation host bounds),
submission/decoder/cancel/cleanup status faults use a separate driver escape,
not `gx_thrown`. Legacy/standalone `gx_fault` continues to abort. Arbitrary
SIGSEGV, abort, allocator catastrophe and hostile worker memory corruption do not
promise cleanup; no signal-based crash recovery is installed.

Main return, source panic/fatal, host fault and reset cancel pending work, wait
every actual ACK and exhaustively clean registrations even after one hook faults.
No background source frames run during retirement. It clears mailbox/run/timer,
context parent/child, pending/source result/panic and task/frame roots. Actual
defer-runner parent/a/b graphs are visited iteratively and detached, including
locals/defers/primitive captures. Direct send/receive/select, mutex, WaitGroup,
nil and sleep registrations detach from externally retained objects; send
payloads clear and stale waiters remain harmless.

Boehm 8.2.8's installed headers are authoritative. `pthread.h` precedes `gc.h`
with `GC_THREADS`. Scanned owner lists/root values preserve source references
while waiting; malloc wire data and `GC_MALLOC_ATOMIC` byte buffers do not hold
source pointers. Whole detached timer/child/cleanup batches use scanned storage
before collection-capable callbacks. See the collector's [C interface](https://www.hboehm.info/gc/gcinterface.html)
and [pinned header](https://github.com/bdwgc/bdwgc/blob/v8.2.8/include/gc.h), checked
2026-10-02. Forced-GC tests prove supported reachability and explicit cleared
fields; conservative collection does not prove object destruction. Native lease
release is explicit, independent of GC/finalizers.

Future adapters must decode declared wire records on the owner, validate lengths
before constructing source values and root partially decoded results. Keys need
an owner/generation registry and opaque wire IDs, with acquired native input
snapshots and explicit output ownership transfer/release. No source descriptor,
`gx_V` graph or opaque key is indiscriminately cloned into wire storage. This
port supplies neither the key registry nor production crypto/HTTP validation.

## Clocks, contexts and stack limits

Host time uses a checked `CLOCK_MONOTONIC` owner epoch, saturating elapsed
nanoseconds and positive deadline arithmetic at signed int64 maximum. Absolute
timers order by deadline then sequence. Due timers run at dispatch boundaries
even while source tasks remain runnable. Empty run queues wait on the mailbox
condition or next real deadline while native work is pending; no virtual
fast-forward or false deadlock occurs. UTC wall-clock capability is independent
and remains unmapped here. A source step that never yields can delay observation.

The actual GCC/glibc POSIX host uses a condition explicitly initialized with
`pthread_condattr_setclock(CLOCK_MONOTONIC)`. Initialization/clock/wait errors are
checked. Mutex predicates/version loops tolerate spurious wake and publication/
ACK/timeout races; no source decode or cancel hook runs with that lock held.
Native timed waits cap to one day and clamp `time_t`/timespec construction before
rechecking absolute deadlines, including int64-max waits. Strict C17 output
gets the POSIX feature declaration from its runtime header.

Contexts capture creation deadlines, inherit the earliest parent, immediately
observe nonpositive/expired parents and register absolute deadlines without
resampling. Child creation and Err/Done observe expiry. Cancellation prunes
parent children/timers/owner contexts. A whole cancellation tree is marked before
native hooks run, so one fault does not skip siblings. Background retains no
owner child/hook registrations. `gx_host_boundary` anchors a positive request-only
timeout before copying/submitting; request expiry does not cancel its source
parent. Nil/foreign source contexts at this generic boundary are distinct declared
validation errors. Nil source constructors retain source-panic behavior; future
production capability boundaries must implement their full declared categories.

Host ordinary source recursion and cooperative frame chains are guarded at 256.
Ordinary returns, panic/defers/recover and driver escapes balance/reset depth;
source fatal bypasses recover. This is a deliberate host implementation depth
limit, not Go dynamic stack growth or recovery after native C stack exhaustion.
Huge individual native/source C frames remain outside that guarantee. Default
sequential/virtual recursion keeps the existing large-stack behavior, verified
by the complete C language suite. Panic Error/String methods render while the
owner remains live; owned report bytes **and length**, including NUL/high bytes,
survive retirement and print without further source execution.

## Evidence and reproduction

Use `GOTOOLCHAIN=go1.25.14` for repository commands; frontend reference tooling
remains unchanged. Native defaults use the actual `driver.ToolEnv` compiler
(observed GCC 11.4.0) and pinned local Boehm 8.2.8. Mandatory ASan/UBSan uses the
retained LLVM 18.1.8 compiler with collector-compatible ASan settings. Test-only
HTTP uses independently observed curl 7.81.0/OpenSSL 3.0.2 subprocesses; no libcurl
headers or production dependency was added. Retained TypeScript browser tooling
follows [its documented pinned prerequisites](typescript-host-operations.md#verification-and-reproduction).

`targets/c/tests/host_operations_test.c` exercises actual runtime registration,
reentrant publication, exact task identity, FIFO/ACK holes, copied inputs/results,
cancel/success/fault ordering, native wake/source progress, controlled deadline
ordering/equality/overflow, expired/nested/request-only contexts, native-thread and
stale callback rejection, repeated pruning, constructor-failure re-entry, exhaustive
cleanup faults and all eight independent blocked roots. Forced collection covers
pending roots, a detached due-timer batch and nested cancellation. A start barrier
precedes the measured runnable-source timer test. All 256 stress results, resumed
source assertions and cleanups complete before main returns.

The shell probe's constructor-failure hook replaces one unique marker only in a
copied runtime file. Emitted integration first runs unchanged virtual source,
then instruments uniquely asserted markers only in copied compiler output.
`tests/integration/c_host_operations_test.go` performs real local HTTP/TLS with
exact normal/release/contact/cancel/trusted-TLS counters across two host entries;
a distinct release request proves source progress. Trusted certificate validation
stays enabled; untrusted TLS never reaches its handler. Actual selectively linked
`task.All` works through the detachable WaitGroup registration.

`tests/integration/c_host_fatal_test.go` proves genuine emitted ordinary/deferred
runtime faults, mutex fatal, plain/custom binary Error panic, ordinary/cooperative
recursion, main return, native fault and genuine deadlock. A native cleanup marker
precedes a parent-held resource ACK gate; the process must remain alive until the
parent releases it. Each case requires fresh operation and final owner markers,
zero native leases and exact stderr/status afterward. The emitted deferred fault
captures the actual >=3-frame parent/a/b graph in scanned test storage, forces GC,
then checks every captured root field clears. Deadlock follows normal response
release and ACK, rather than canceling native work to create the deadlock.

The test-only adapters omit the production URL/header/body bounds, redirect,
authentication and typed HTTP contracts. They create no HTTP mapping, SDK library
or KAS acceptance. All logs/launchers/hashes/historical failures are ignored under
`out/c-host-operations/`; final terminal results are recorded after the frozen
broad replay. No Phase 4 or whole delivery completion is claimed.


### Terminal frozen verification (2026-10-02)

The final 44-file freeze was captured at `2026-10-02T16:44:00.819396+00:00`.
Launcher PID 3472847 / exec session 92377 completed without restarting; exact
commands, individual statuses and logs remain in `out/c-host-operations/`.

| Check | Terminal evidence |
| --- | --- |
| Focused native, ASan/UBSan and emitted lifecycle | Passed; `final-focused.log`, including all 11 fatal cases, 512 recovery iterations, real HTTP/TLS and emitted TaskAll. |
| Full C language | Passed in 43.711 s; `final-c-language.log`. |
| All-seven contracts | Passed in 148.225 s; `final-contracts.log`, including byte/collector defaults and mandatory sanitizers. |
| Internals, all-target corpus and full integration | Passed; `final-internals-corpus-integration.log`; corpus 56.498 s, integration 380.577 s. |
| Actual browser and TypeScript typecheck | Passed; Chromium 147.0.7727.15, Playwright 1.58.2, esbuild 0.25.12; `final-browser.log`, `final-typecheck.log`. |
| C virtual harness | Passed 179/179; `final-c-harness.log`. |
| Catalog, generated freshness and frozen hashes | Passed; `final-catalog.log`, `final-freshness.log`, `final-production-verified.log`. |
| Accepted targets and source preservation | C#/Java/Rust/accepted TypeScript hashes passed. Original baseline/Python checks failed solely on a regenerated ignored Python bytecode cache; source-only supplements passed 1453 unchanged source files and all 156 Python source files. |
| Local documentation links and whitespace | Passed; `final-local-doc-links.log`, `final-whitespace.log`. |

The launcher itself returned 2 because the original preservation manifests
included generated `__pycache__` files. Its failing manifests/logs are retained;
`preservation-cache-supplement.json` lists the exact cache exclusions. No source
hash was replaced to hide a production mismatch. This paragraph/table is a
terminal-results-only documentation supplement after the frozen broad replay;
`documentation-supplement.json` records its pre/post hash, while the original
44-file freeze and the unchanged 40-file runtime/test/metadata hashes remain
separate evidence. This is bounded C host verification evidence, not a library,
production adapter or real-KAS delivery result.

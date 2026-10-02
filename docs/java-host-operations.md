# Java pending host operations

This bounded Phase 4 port provides a generic JVM worker/future mailbox lifecycle
and an explicit monotonic entry for compiler-generated cooperative executables.
Production `lib.http.do`, crypto mappings and generated SDK libraries remain
unavailable. The Java capability mappings and GCE006 library gate remain closed.
The remaining four host ports and full seven-target library work are still required.

## Entry and source ownership

Cooperative output exposes `Main.runHost()`, which calls
`TaskSpawn.runMainHost` with the actual compiler entry-frame factory. It blocks
its calling driver thread until main finishes and native cleanup is acknowledged.
Its clock uses an owner-local `System.nanoTime` epoch. Default `Main.main`,
`sh run.sh`, and runtime harnesses retain deterministic virtual time, byte-exact
output, source positions, panic/deadlock reporting and exit status 2.

The executable ABI retains static source globals, `TaskSpawn.sched` and panic
bindings. Every executable entry, harness entry and reset reserves the same
atomic entry guard before owner installation; concurrent entry/reset rejects
without changing the active owner. Source frames execute only on the owner
thread. Runtime transitions check that thread, and native callbacks receive
only tokens rather than source task/frame references. Retirement restores a
usable default panic binding.

A real emitted two-call test prints `init 1`, `main 2`, then `init 3`, `main 4`.
Initialization runs again against persistent source state; there is no package
init guard. Sequential executable reuse is neither initialize-once library
semantics nor independent instance globals. Future library work must own source
globals, clients, keys, initialization, calls and runtime bindings per instance.
The exported-library acceptance boundary remains
[the SDK requirements](../../sdk/docs/library-requirements.md).

## Mailbox, completion and cleanup

The driver alone changes source frames, result vectors, contexts, channels,
timers and run queues. Native workers/futures own copied inputs and native
resources; they publish records and wake a synchronized owner mailbox. A
`HostToken` retains mailbox identity and numeric operation/task IDs only, never
the owner, input snapshots or source roots. Mailbox identity acts as its unique
owner generation. Closed-owner and foreign-generation tokens cannot resume a
new owner. The mailbox also validates each operation's registered task ID before
accepting either publication or cleanup acknowledgement. An invalid task ID
cannot consume the live operation, poison its first result, or acknowledge its
resources; a later valid completion still applies and cleans exactly once.

`registerHost` installs the operation and parks its task before submission.
Synchronous/reentrant completion follows the same mailbox path. The first
terminal record for an operation is retained; duplicates are discarded. Records
apply only after their separate `acknowledgeCleanup` signal. Dispatch appends
applicable tasks FIFO in publication order. Stale records and ACKs are discarded;
record/live/ACK sets are bounded by registered operations and cleared on retirement.
Completed background source tasks are removed promptly.

`launchHost` attaches native `CompletionStage` settlement handlers immediately.
Unexpected submission throws and future rejection become `HostFault`, which
bypasses source panic/recover. The work stage must settle only after its own
resource cleanup. A canceled `Future` reaching a terminal state does not prove
worker/body/input/key release. Adapters with independently delayed cleanup must
register manually and publish the cleanup ACK only after that cleanup finishes.
Declared transport and validation failures belong in ordinary result records.

At dispatch, cancellation observed before ordinary result application wins;
committed success remains final. An applicable adapter fault beats cancellation.
Decoder/cancellation-bridge and registration-cleanup faults are also host faults,
and retirement still releases every other registration.

Wire snapshots recursively copy `byte[]`, object arrays, lists and string-keyed
maps with primitive/string leaves. They reject cycles, source `Box` descriptors,
closures, frames and opaque native handles. This is a restricted transport
representation rather than the final SDK result API. `registerHost` accepts a
driver-side decoder: later adapters must encode declared errors as plain records
and construct their source descriptors on the owner. Native keys need an owner
registry and opaque wire IDs, with explicit transfer of output ownership and
independently acquired input snapshots retained through cleanup. No key registry
or production crypto adapter is implemented here. Unsigned source bytes retain
Java's existing normalized long values and native `byte[]` storage.

Main return, source panic, host fault and reset retire the owner. Shutdown marks
it retiring, requests native cancellation, and waits for every pending cleanup
ACK before releasing operation registrations. It then clears mailbox records,
run queues, timers, contexts and all task/frame/result/panic roots without
executing more source tasks. Cleanup continues after individual cancellation,
disposer or task-cleanup hooks throw. A broken adapter that never acknowledges
cleanup prevents safe return; the runtime cannot invent a release acknowledgement.
Fatal panic/deadlock/actual stack overflow/shared mutex-fatal exit 2 occurs after
retirement, rather than inside a source step.

Direct send/receive, blocked select, mutex and WaitGroup registrations detach
from retained external channel/sync objects during retirement. Select cleanup
retains only registered waiters/channels rather than the evaluated send cases.
Retained stale waiters lose their task and send payload and safely ignore later
callbacks. Nil channels, empty select and sleep also drop owner roots. Channel
buffers now use a null-capable deque so buffered nil pointers preserve ordinary
source behavior, FIFO order, select and harness representation.

## Clocks and contexts

`System.nanoTime` supplies elapsed monotonic nanoseconds; UTC wall clock remains
a separate capability. Timers are ordered by absolute deadline then registration
sequence. Due real timers dispatch even with runnable source tasks. When no task
is runnable, the owner waits on mailbox publication or its next real deadline.
Pending native I/O prevents false deadlock and virtual fast-forward. Source
execution remains cooperative: a single source step that never yields can delay
deadline observation.

Context deadlines start at creation, inherit the earliest parent deadline and
immediately observe nonpositive/expired parents. They register the captured
absolute deadline with `addTimerAt`, preserving it when consecutive clock samples
differ. Positive duration arithmetic saturates at `Long.MAX_VALUE` instead of
wrapping. Cancellation removes parent/child/timer/hook registrations; retirement
removes remaining context roots. Source nil contexts remain source nil-dereference
panics, while foreign source owners are host faults. Future capability adapters
must classify invalid boundary inputs as declared validation errors.

`StdContextErr.Boundary` provides a generic request-only timeout captured before
copying/submitting native work. It inherits the source deadline without canceling
the parent on request-only expiry. Its hooks and absolute timer detach at cleanup.
It is lifecycle plumbing, not an HTTP contract implementation.

## Evidence and reproduction

Compiler commands use `GOTOOLCHAIN=go1.25.14`; actual JVM builds/runs resolve the
pinned local JDK 21 through `driver.ToolEnv`. The existing frontend reference
Go 1.27.1 selection, profiles, references and lockfiles are unchanged. Evidence
is ignored under `out/java-host-operations/`.

`targets/java/tests/HostOperationsTest.java`, launched by
`tests/contracts/java_host_operations_test.go`, runs the actual JVM runtime:
registration/reentrant completion, FIFO/out-of-order/duplicate/stale ownership,
byte snapshots, cancellation/success/fault precedence, driver decode faults,
monotonic deadlines with drift/equality/overflow/order, context/hook pruning,
nil/foreign context categories, native wake plus source progress, future rejection,
exhaustive cleanup faults and independently retained blocked roots. It tests
256 sequential operations and a real 256-operation concurrent backlog, asserting
every resumed result and cleanup before main returns.

Retirement gates retain copied input snapshots and mock input/key ownership
counters after native cancellation is requested. A separate `CompletableFuture.cancel(true)` terminal-state assertion is a lifecycle model, not a real crypto handle or production transport cleanup proof. The test establishes that the driver is actually
WAITING in `Mailbox.await` from `Scheduler.shutdown` before releasing cleanup.
Fresh fatal subprocesses hold native cleanup behind the parent's stdin gate,
then verify exact stderr bytes, exit 2 and exact unique cleanup-marker contents.
The overflow subprocess causes genuine recursive JVM stack overflow.

`tests/integration/java_host_operations_test.go` executes unchanged emitted
virtual behavior first, including the ordinary buffered nil `*int` regression.
It then adds visibly test-only adapters to copied output using two unique marker
assertions, and drives actual emitted frames through `Main.runHost`. Independent
local HTTP servers gate body release through a second real request, proving source
progress while the first operation remains pending. Source cancellation executes
through emitted context code. Exact request counters prove both repeated entries
contacted the real servers. TLS tests trust the independent certificate explicitly
and verify default untrusted TLS rejection never reaches its HTTP handler.

The test-only `HttpURLConnection` adapter is a worker bridge, not production
`lib.http.do`. Cancellation requests disconnect on a native thread and cleanup
can await the bounded ten-second read timeout while disconnect contends with
body reading. The observed two-call integration took approximately 26 seconds.
This proves cancellation precedence and resource retention through ACK, without
claiming prompt production HTTP cancellation or the full transport/redirect/
header/body/error contract. A later production adapter must satisfy that contract
before a capability mapping is added.

The initial focused runtime/byte replay passed. The expanded replay passed runtime
and byte tests but failed an incorrect virtual fixture oracle: the ordinary
123ns sleep returns before the independent 124ns progress sleep. The corrected
emitted replay passed; its real HTTP gate requires progress before transport return.
The original failure remains in `focused-expanded.log`.

Every frozen command below ended with status 0; no required tests were skipped.
The retained original launcher was PID 2531591, exec session 69698. It completed
without a restart or discarded observation timeout. Pinned Java was Temurin
21.0.12.1+1; `versions.log` records the actual driver-selected binaries.

| Command | Frozen evidence |
| --- | --- |
| `go test -v -count=1 ./tests/contracts` | Passed all seven harnesses, byte defaults, Java lifecycle/fatal subprocesses, mandatory C ASan/UBSan and actual TypeScript typecheck; 138.474s, `contracts-frozen.log`. |
| `GOALCHEMY_TEST_TARGETS=java go test -v -count=1 ./tests/language` | Full Java language passed; 55.763s, `java-language-frozen.log`. |
| `go test -v -count=1 ./internal/... ./tests/corpus ./tests/integration` | All internals and all-target corpus passed (70.782s); full integration passed (219.427s), including actual emitted Java HTTP/TLS/reuse/nil pointer (25.770s), Go HTTP and TypeScript Node/Chromium; `compiler-corpus-integration-frozen.log`. |
| `node targets/typescript/tests/host_operations_browser.mjs` | Actual Chromium 147.0.7727.15 passed the retained portable host suite and genuine fetch/source progress; `browser-frozen.log`. Pinned ignored Playwright/esbuild tooling is described in [the TypeScript reproduction instructions](typescript-host-operations.md#verification-and-reproduction). |
| `out/java-host-operations/goalchemy test -target java` | Passed 179/179 virtual contract cases; `java-harness-frozen.log`. |
| `out/java-host-operations/goalchemy spec validate` | Passed 13 types, 94 functions, seven targets; `catalog-frozen.log`. |
| `out/java-host-operations/goalchemy spec generate -check` | Passed all 503 current generated files; `freshness-frozen.log`. Generation followed the final production freeze; `catalog-freeze.log` records the build/validation/generation. |
| `git diff --check` and changed-document local link checks | Passed; `whitespace-frozen.log`, `documentation-final.log`. |

The scripts, terminal status files, exact source hashes and final outcome inventory
remain under `out/java-host-operations/`. Earlier focused logs are retained as
historical evidence; they are not substituted for the frozen final replay.

Root review subsequently reproduced a same-owner, wrong-task token consuming a
live operation before its cleanup. The narrow correction above adds an
operation-to-task mailbox identity map and validates records before owner-side
removal. The durable JVM test now sends a foreign fault and ACK, then a valid
result with a foreign ACK, and finally the valid ACK: only the last resumes the
task and performs its single cleanup. The original failed copied-runtime probe
is retained in the SDK's ignored `.local/root-java-foreign-mailbox.log`.
The post-correction lifecycle/fatal replay passed (1.446s) in
`.local/root-java-host-foreign-repair.log`, and actual emitted Java frames,
HTTP/TLS, cancellation and repeated initialization passed (26.366s) in
`.local/root-java-host-foreign-emitted.log`. The original frozen hashes and logs
above remain evidence for the preceding implementation, rather than hashes of
this subsequent correction. The current C# assignment's final all-target checks
and regeneration must include the corrected Java sources.

This completes the bounded generic Java lifecycle implementation and evidence,
subject to root acceptance. It does not complete Phase 4, production target
adapters, exported SDK libraries or full SDK parity.

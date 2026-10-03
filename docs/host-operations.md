# Host operation lifecycle

This records the bounded Phase 4 generated Go design. The generic TypeScript
Node/browser lifecycle and explicit Promise executable entry now have a separate
[implementation and evidence](typescript-host-operations.md). Production TypeScript
HTTP/crypto adapters, other target ports and exported libraries remain required. Java now provides a bounded [native mailbox lifecycle and explicit executable entry](java-host-operations.md); its production adapters and exported libraries remain gated. C# now provides a bounded [Task/mailbox lifecycle and managed source-recursion guard](csharp-host-operations.md); production adapters and SDK libraries remain gated. Python now provides a bounded [Condition/Future mailbox lifecycle and explicit synchronous host entry](python-host-operations.md); production adapters and SDK libraries remain gated. Rust now provides a bounded [native mailbox/resource-ACK lifecycle and dedicated monotonic entry](rust-host-operations.md); production adapters and SDK libraries remain gated. C now provides a bounded [native wire mailbox/resource-ACK lifecycle and explicit monotonic executable entry](c-host-operations.md); production adapters and SDK libraries remain gated. Native HTTP transport
validation stays in `lib/http`; a generated adapter must call that implementation.

## Ownership and transitions

The scheduler alone mutates tasks, frames, result vectors, context state,
cooperative channels and timer/run queues. An operation token identifies the
creating scheduler, its generation, an operation sequence and its task. In Go,
the mailbox identity is the unique owner/generation; the token holds that
mailbox plus sequence/task IDs, and never retains frames or request buffers.
A host worker owns copied inputs and native resources until it has finished cleanup.
It transfers an immutable result or fault record into the owner's mailbox and
signals its wake channel. It never executes source code.

Registration installs the pending record and parks its task before submission.
Submission may finish synchronously or call back before returning; both follow
the same mailbox path. The driver drains completions at dispatch boundaries.
The first applicable record removes the pending registration, releases adapter
registration cleanup, assigns the result vector and appends the task to the FIFO
run queue. Duplicate, foreign and retired-generation records cannot resume it.
Adapters must acknowledge finished resource cleanup with their terminal record;
cancellation does not report a result while resources are still being used.

Source context cancellation marks the context first and invokes nonblocking
native cancel hooks. Cancellation observed before a completion is applied wins
over a queued success, returning zero outputs and the context error. Once a
success is applied, later cancellation cannot rewrite that result. Thus races
are determined by driver observation order, rather than host thread scheduling.
Host transport/validation errors are declared error results. Adapter panics are
host faults propagated by the driver outside source panic/recover handling.
An applicable host fault wins over ordinary cancellation. Only the first
terminal record is applicable; duplicate or retired-owner fault records are
discarded just like stale ordinary completions.

Entry return retains existing executable behavior: background source tasks do
not keep main alive. Shutdown cancels pending native work, waits for worker
cleanup, closes the mailbox and drops retained records. Reset retires the old
owner before installing a fresh scheduler. A late token cannot revive a task in
another instance. Mailboxes close and discard their queues at retirement, and
tokens never retain the owner itself. Source panic and host fault also retire
their owner; executable fatal source panic cleanup runs before `os.Exit`.

## Clocks and contexts

`RunMain` and conformance `Await` retain deterministic virtual time. Generated
programs with a crypto, HTTP or callback contract in their emitted linked
contract set select their target's host entry (`RunMainHost` in Go, `runHost`
elsewhere); this selection belongs to the emitted entry, never an
environment variable or process-wide
clock switch. Host mode uses a scheduler-local monotonic epoch. Sleep and context
timeouts start when called, including source work before request submission.
Timers are ordered by deadline then registration sequence and due timers run
at driver dispatch boundaries, including when other tasks are runnable. A
driver without runnable tasks waits on its mailbox wake channel or the next
real deadline. Pending I/O prevents false deadlock and virtual fast-forward.
`lib.clock.Unix` remains an independent real UTC observation.

Contexts retain their owner and earliest inherited deadline. Native request
contexts receive that absolute deadline, preserving earlier parent timeouts,
and cooperative cancellation invokes native cancellation. Context `Done` and
`Err` are updated by the owner only. Canceled child and deadline registrations
are removed from their parent/timer queues, and operation cancel hooks detach
when cleanup finishes. Nil contexts produce declared HTTP validation errors.
Positive timer arithmetic saturates at the clock's maximum rather than wrapping
to an expired deadline. Context errors use standard Go context sentinels, so
source identity and `errors.Is` work for wrapped native transport errors too.
The required HTTP timeout is anchored by native deadline creation at the source
boundary before copying/submitting work. Worker scheduling delay consumes that
timeout. Request-only expiry returns a deadline error without canceling its
source parent. Coarse declared input bounds reject through native error behavior
before allocating snapshots; native `lib/http.Do` remains authoritative for
URL/header/transport validation, TLS, redirects and body limits.

Shared declarations/default cases remain version 1.0.0 during pre-release
Phase 4. The explicit host entry extends clock selection and context timeout
metadata is `controlled`; it does not change virtual conformance fixtures.

## Future target and library work

Each exported library instance needs an explicit scheduler owner with its own
clock, mailbox, shutdown and generation, plus serialized driver entry. The
current executable global runtime ABI does not establish library isolation and
the library gate remains in place. Browser ports use promise completions and
AbortController; threaded hosts use immutable completion queues. C must pair
every retained handle/buffer with an explicit release. All ports preserve the
same registration-before-submission, cancellation precedence and cleanup
acknowledgement rules. Async crypto must acquire and retain native key snapshots
before submitting, allow Close to reject new acquisitions without revoking an
existing snapshot, and release snapshots before reporting cancellation.

Port shapes and remaining implementation boundaries:

| Target | Completion, cancellation and retention |
| --- | --- |
| TypeScript Node/browser | Generic Promise/mailbox lifecycle, cleanup ACK, portable host boundary and explicit serialized executable driver are implemented and browser-tested; see the TypeScript evidence. Production fetch/crypto adapters and isolated library globals remain gated. Test-only fetch is not the shared HTTP implementation. |
| Java | Generic native worker/future mailbox lifecycle, separate cleanup ACK, real monotonic entry and blocked-registration retirement are implemented and JVM-tested; see the [Java evidence](java-host-operations.md). Production HTTP/crypto and isolated library globals remain gated. |
| C# | Generic native Task/mailbox lifecycle, explicit publication FIFO, separate cleanup ACK, real Stopwatch entry, blocked-registration retirement and managed source-recursion guard are implemented and CLR-tested; see the [C# evidence](csharp-host-operations.md). Production HTTP/crypto and isolated library globals remain gated. |
| Python | Generic Condition/Future mailbox lifecycle, separate resource ACK, synchronous monotonic executable entry, blocked-root retirement and managed recursion handling are implemented in CPython; see the [Python evidence](python-host-operations.md). Production adapters and SDK libraries remain gated. |
| Rust | Generic Send-wire mailbox, true resource ACK, dedicated monotonic executable owner, traced operation roots, blocked/frame-cycle retirement and generated recursion guards are implemented; see the [Rust evidence](rust-host-operations.md). Production adapters and SDK libraries remain gated. |
| C | Generic native wire mailbox, exact operation/task resource ACK, monotonic serialized executable entry, scanned source roots, blocked/defer-frame retirement and generated recursion guards are implemented; see the [C evidence](c-host-operations.md). Production adapters and SDK libraries remain gated. |

Every port needs monotonic deadline anchoring from context/request creation,
first-terminal-record application, stale-owner rejection and the same
fault-versus-cancellation distinction. A future library owner must also contain
its source globals and blocked-task registrations, dropping those roots during
shutdown without running background source frames. Existing executable globals
are not a supported overlapping-library-entry API.

## Evidence

`targets/go/runtime/host_operations_test.go` executes the actual runtime with
synchronous callbacks, pending-without-run-queue waits, FIFO/out-of-order/duplicate
completion, owner isolation, deterministic cancel/success/fault precedence,
retirement, cleanup counts, timer ordering/overflow and repeated actual local
HTTP requests. Its subprocess fatal test proves worker cleanup precedes the
real executable `os.Exit`; its delayed-submission test waits for the anchored
native deadline before starting the real HTTP worker. Race runs include bounded
256-operation stress and repeated request/cancellation-hook cleanup.

`tests/integration/go_host_http_test.go` compiles ordinary source through the
real driver, then runs emitted Go with the race detector against independent
stdlib HTTP/TLS servers. Server synchronization proves concurrent source
cancellation during delayed headers/body reads. Cases cover earlier/nested and
already-expired/canceled contexts, pre-submission source sleep, function-value
dispatch, mandatory timeouts, copied inputs, independent responses, background
shutdown, redirects/non-2xx, body/header/gzip limits, TLS rejection and nil context
comparisons in both directions, including a source alias.

Evidence logs are ignored under `out/go-host-operations/`. Native Go capability
and generated Go platform-health probes supplement these scheduler checks; a
health GET alone does not prove exported SDK/KAS interoperability. All commands
below use `GOTOOLCHAIN=go1.25.14`, from the repository root unless stated otherwise.

| Verification | Result and log |
| --- | --- |
| `go test -race -count=10 ./targets/go/runtime` | Passed, including actual fatal subprocess and anchored-request-timeout tests; `runtime-anchored-timeout.log`. |
| `go test -race ./lib/context ./lib/crypto ./lib/encoding ./lib/http ./lib/clock ./targets/go/runtime` | Passed; `native-race-final.log`. `lib/context` has no standalone package tests; its runtime/source bridge is covered above. |
| `go test -v ./tests/contracts` | Passed with no skips, all seven harnesses and byte defaults, including mandatory C sanitizer checks; `contracts-final.log`. |
| `GOALCHEMY_TEST_TARGETS=go go test ./tests/language ./tests/corpus` | Passed; `go-language-corpus.log`. Language uses the Go filter; corpus runs all seven targets. |
| Compiler/frontend/driver/subset/link/lowering/emitter/specgen plus full integration packages | Passed in the frozen final replay, including actual emitted HTTP; `compiler-final.log`. The earlier successful run remains in `go-host-operations-compiler.log`. |
| `out/goalchemy spec validate` | Passed: 13 types, 94 functions, seven targets; `catalog-final.log`. |
| `out/goalchemy spec generate -check` | Passed: 503 current generated files; `freshness-final.log`. The new Go HTTP mapping adds one file to the prior 502. |
| `out/goalchemy test -target go` | Passed: 180/180 cases (179 existing plus HTTP invalid-boundary validation); `harness-go-final.log`. |
| SDK capability primitive probes, ordinary and emitted Go with `-race` | Passed; `sdk-primitives-native.log`, `sdk-primitives-generated.log`. |
| SDK existing HTTP health probe, ordinary and emitted Go with `-race` | Passed against the existing local platform `/healthz`; `sdk-http-native.log`, `health-run.log`. |

The intermediate full contracts run failed generated freshness because the
runtime changed after generation. Its failed evidence is retained as
`contracts-failed-stale-task-hash.log`; every behavior test passed in that run.
Regeneration and the frozen full replay above passed. No all-target host-operation,
browser, exported SDK library or Phase 4 acceptance is claimed by this document.

# TypeScript portable host operations

This bounded Phase 4 work implements generic pending-host lifecycle and an
explicit Promise entry for cooperative executable programs in Node and actual
browsers. It does not implement production `lib.http.do`, crypto capabilities,
exported SDK libraries, persistent clients, or two instances of source globals.
The TypeScript capability and GCE006 library gates remain closed. Phase 4 remains
open for the other host ports and exported-library work.

## Entry and portable host boundary

Cooperative output contains `program.ts` with the actual generated frames and
source globals, its `program.ts.map`, the Node executable `main.ts`, and portable
`host.ts`. The Node wrapper installs `rt/types/node_host.ts` and calls the
synchronous generated runner. Sequential output retains its existing `main.ts`
and map. `node main.ts` preserves virtual conformance time, seed handling,
byte-exact output, panic reports, and exit status 2. The Node adapter writes
`Uint8Array` bytes with `writeSync`; no UTF-8 text conversion touches arbitrary
Go string bytes.

Importing `host.ts` does not execute source initialization or main. Its
`runHost(host)` returns a Promise and executes the compiler's source init/entry
frame through `runMainHost`. Each invocation drives the compiler init/entry against existing module-global
source state. The actual retained two-call probe prints `init 1`, `main 1`, then
`init 3`, `main 3`: its source init executes again and its counter persists
(`reuse-result.log`). The probe output has no generated package-init guard.
Sequential reuse can rerun initializers against persistent state; it creates
neither fresh globals nor a persistent client API. This executable behavior
does not implement initialize-once exported-library instance semantics. Unsupported overlapping calls, synchronous
entry, and reset during an active host drive reject before changing the active
owner. The current runtime still has a global `sched` and recover binding.
A future library must move source globals, exported call state, clients, runtime
bindings, background registrations and key registries into an explicit instance;
owner-isolated operation mailboxes alone cannot provide that isolation.

`RuntimeHost` supplies monotonic milliseconds, cancellable alarm scheduling,
byte stdout/stderr, executable failure, and a seed. The portable default uses
`performance.now` and timers; output and executable failure require an explicit
adapter. A supplied host owns ordinary source output as well as failure output
for the duration of the drive, and the prior output adapter is restored afterward.
The recover binding is reset after async retirement. The browser dependency
graph uses no Node import, `Buffer`, `process`, or Node compatibility shim.
`types/utf8.ts` uses `TextEncoder` only when encoding an actual host UTF-16 string;
existing arbitrary-byte Go string and rune decoding is unchanged.

## Ownership, completion and cleanup

The driver owns all source tasks, frames, channels, contexts and timers.
Registration installs an operation and parks its task before adapter submission.
A token contains its owner/generation mailbox identity and numeric operation/task
IDs. The mailbox records the task ID for each live operation and checks it
before both result/fault publication and cleanup ACK. A token for the wrong task
cannot release another task's valid queued result. It never contains a scheduler,
source frame, or request/input reference.
Synchronous/reentrant callbacks and Promise settlements use the same mailbox.
They enqueue a record and signal the driver; they never resume source themselves.

Completion snapshots structured-cloneable result data, including nested arrays
and bytes, before publication. The adapter cannot mutate the queued snapshot.
This is a wire-data boundary, not a representation for source error descriptors
or native key objects: structuredClone rejects functions in source Box descriptors.
Later capability adapters must encode declared errors into plain records, then
construct the source Box on the owning driver. Native keys require an owner key
registry: publish a cloneable opaque ID, transfer the output key's ownership to
that registry, and decode the ID on the driver. Input key acquisitions must
retain their native snapshots until operation cleanup; output handles have a
separate library-instance/Close lifecycle. This assignment implements no key
registry or crypto mapping and does not pass CryptoKey objects through cloning.

`acknowledgeCleanup` is separate from completion. The adapter calls it only after
native work stops using its retained input/key snapshots and releases its native
resources. A terminal record cannot apply until its matching acknowledgement.
`launchHost` attaches success and rejection handlers immediately, converts an
unexpected throw/rejection to `HostFault`, and acknowledges settlement cleanup.
Its work Promise must settle only after native cleanup. Adapters that need delayed
cleanup can register manually and acknowledge explicitly after their cleanup gate.
Declared transport/validation failures must be ordinary result records, not
unexpected Promise rejections. A broken adapter that never acknowledges resource
cleanup prevents shutdown from finishing; the runtime cannot safely invent an ACK.

At each dispatch boundary, the first applicable record retires the operation,
runs registration cleanup, assigns its result, and queues its task FIFO. Duplicate,
foreign-owner, retired-generation and late-fault records are ignored. Late/duplicate
ACKs are discarded too; terminal bookkeeping does not grow with callbacks after
retirement. Cancellation observed by the driver before application overrides an
ordinary queued success; an applied success is final. An applicable adapter fault
beats ordinary cancellation and bypasses source panic/recover.

Main return, source panic and host fault cancel pending adapters and await their
ACKs. Retirement discards queued records and clears runnable/blocked task frames,
result vectors, timers, context hooks/children and cleanup registrations without
executing additional source tasks. Cleanup exceptions do not prevent other roots
from being released. Completed background tasks are removed promptly. Synchronous
virtual reset also retires its previous owner; a live asynchronous owner requires
its Promise shutdown path. Late callbacks cannot revive a new owner. Fatal source
panic/deadlock/stack overflow reporting happens after cleanup, including before
real Node `process.exit(2)`.

## Clocks and contexts

`runMain` and `runIsolated` keep deterministic virtual time and default fixtures.
`runMainHost` has a per-owner monotonic epoch, processes due timers even when
source tasks are runnable, and yields a real host turn between dispatches so
Promise settlements, host timers and I/O can progress. Pending host I/O keeps an
empty run queue from being reported as deadlock; real time never fast-forwards.
`lib.clock` is a separate wall-clock capability and is not implemented here.
The clock choice follows [High Resolution Time's monotonic
clock](https://www.w3.org/TR/hr-time-3/). Browser alarms follow
[HTML timer rules](https://html.spec.whatwg.org/multipage/timers-and-user-prompts.html#timers),
including minimum delays for nested timers; real-time deadlines are observed at
dispatch boundaries and do not promise nanosecond alarm precision. These primary
standards were checked on 2026-10-02.

Contexts inherit the earliest absolute deadline at creation. Nonpositive or
expired deadlines cancel immediately. Child creation and Err/Done observe expired
inherited deadlines even before a queued timer callback dispatch. Absolute timer
registration preserves the sampled deadline exactly; positive duration addition
saturates at signed 64-bit clock maximum. Cancellation prunes parent children,
timers, hooks and owner registrations, including when a hook throws. Foreign
context-owner use raises a host fault; nil context operations raise source panic.
A generic request-only timeout can be anchored before snapshot/submission without
canceling its parent; this is lifecycle evidence, not a production HTTP mapping.

## Verification and reproduction

All commands run from the Goalchemy repository root, except the stated typecheck.
Compiler/test commands use `GOTOOLCHAIN=go1.25.14`. The frontend reference selection
remains the existing Go 1.27.1 repository/tool environment; no reference/profile/
lockfile was changed. Logs live in ignored `out/typescript-host-operations/`.

Pinned test tooling is installed only under ignored storage, with no dependency
lock changes:

```sh
npm install --prefix out/typescript-host-operations/tooling --no-package-lock --no-save playwright@1.58.2 esbuild@0.25.12
```

The browser harness resolves `GOALCHEMY_CHROME`, then `google-chrome`, `chromium`,
or `chromium-browser` on PATH, then Playwright's installed default browser.
If a browser is absent, install Playwright's pinned Chromium into ignored storage
and set `PLAYWRIGHT_BROWSERS_PATH` consistently for installation and test execution.
The observed environment is Node v24.15.0, TypeScript 6.0.3, npm 11.12.1,
Playwright 1.58.2, esbuild 0.25.12, and Chromium 147.0.7727.15. Version outputs are
retained in `versions.log`.

The frozen replay passed with terminal exit 0 for every required command and
no failed/skipped tests. Contracts took 238.613 seconds; full TypeScript language
102.006 seconds; corpus 123.137 seconds; full integration 252.668 seconds. The
C sanitizer test ran and passed; catalog has 13 types/94 functions/seven targets
and all 503 generated files are current.

| Command | Evidence |
| --- | --- |
| `node targets/typescript/tests/host_operations.test.ts` | Actual Node runtime, controlled Promise gates, ACK/root bounds, snapshots, owner isolation, cancellation/fault ordering, rejection consumption, real deadlines, 256 counted source resumes/cleanups, and real fatal Node subprocesses with distinct per-mode markers and exact cleanup contents. `runtime-subprocess.log`, final contracts log. |
| `node targets/typescript/tests/host_operations_browser.mjs` | Same portable suite in genuine Chromium; all bundle inputs checked for Node dependencies; genuine browser fetch echo with unrelated source progress. `browser-frozen-final.log`. |
| `GOTOOLCHAIN=go1.25.14 go test -v ./tests/integration -run TestGeneratedTypeScriptHostOperations -count=1` | Actual compiler output, portable import without execution, emitted frames via Promise entry in Node and Chromium, nil/earliest/expired contexts, real source sleep, HTTP and trusted local TLS. `emitted-browser-initial.log`, final integration log. |
| `(cd targets/typescript && npm run typecheck)` | Full runtime/harness/test typecheck; final contracts log also executes it. |
| `GOTOOLCHAIN=go1.25.14 go test -v -count=1 ./tests/contracts` | All seven target contracts, every accepted byte default, mandatory C sanitizer checks, TypeScript lifecycle and generated freshness. `contracts-frozen-final.log`. |
| `GOALCHEMY_TEST_TARGETS=typescript GOTOOLCHAIN=go1.25.14 go test -v -count=1 ./tests/language` | Full TypeScript language suite. `language-frozen-final.log`. |
| `GOTOOLCHAIN=go1.25.14 go test -v -count=1 ./tests/corpus ./tests/integration ./internal/...` | Corpus on all targets, full integration and internal regressions. `compiler-corpus-integration-frozen-final.log`. |
| `out/goalchemy spec validate` / `out/goalchemy spec generate -check` | Catalog and generated freshness. `catalog-frozen-final.log`, `freshness-frozen-final.log`. |

The emitted integration's sentinel Sleep instrumentation is a visibly test-only
adapter installed in a copied generated runtime. It verifies exactly one injection
boundary and actual independent server request counts in Node and Chromium. The
source program and all its resumable frames are compiler output; they are not
handwritten replacement frames. Real sleeps and contexts still use production
runtime paths. The independent HTTP/TLS transports and browser fetch helper are
not implementations of `lib.http.do`: they omit shared bounds, header fidelity,
redirect and credential policy, and production capability error mapping.

A second post-root-repair contracts replay failed only generated freshness after
late fatal-path changes (`contracts-postroots-failed-stale-before-fatal-regeneration.log`);
all its behavior checks passed. Regeneration and the separate frozen replay are
required final evidence. No other-target production file or accepted byte behavior
was changed by the host assignment.

The initial reuse probe assumed guarded initialization and failed its expectation
(`reuse-failed-guard-assumption.log`). Its actual compiler output establishes
repeat init against persistent globals; corrected `reuse-result.log` passes the
observed behavior without claiming library initialize-once semantics.

The first broad replay completed with contracts, TypeScript language, corpus,
integration and internal checks passing before the blocked-registration/fatal
repairs. Those logs are retained as `*-prefix.log`. Its browser run failed the
load-sensitive `ticks > 0` progress assumption; the failed log is retained as
`browser-prefix-failed-load-sensitive-progress.log`. A source-step progress barrier
now starts the runnable worker before the measured deadline is created. Real sleep
and deadline evidence remains separate from controlled cleanup-order gates.

Root's retained-root probe found one send waiter, one receive waiter and a retained
1 MiB send payload after scheduler task roots were cleared. The repaired lifecycle
now installs removable direct channel waiters, clears select losers, and prunes
mutex/WaitGroup registrations. The durable suite retains actual stale waiter
references, checks task/value fields clear, and calls their completion methods
after retirement to prove results/panics/queues cannot revive. Root's reproduction
logs are `sdk/.local/root-ts-blocked-roots-{initial,fixed}.log`. A further review
found source mutex fatal reporting needed the same driver cleanup path as panic;
that path now throws HostFatal until cleanup ACK and is tested before real exit2.
Final post-repair logs below are a separate frozen replay, not the pre-fix evidence.

Historical failures: initial typecheck found `fatal` needed an explicit return of
the host's never-returning failure; initial Node execution rejected TypeScript
parameter properties under strip-only mode (removed); initial emitted integration
mistakenly treated Go println as stdout and rejected its correct stderr bytes
(`emitted-initial.log`). Corrected emitted runs preserve Go output semantics.
The first stress test could finish main before all workers resumed; final stress
requires 256 accepted source results and cleanups. Initial fixed-delay cancellation
observations were replaced with explicit cancellation gates. These failures and
review fixes do not count as final verification successes. Intermediate blocked-root
fixture typechecks also caught nullable child/numeric assertion narrowing; final
typecheck (`typecheck-frozen-final.log`) corrects those test-only declarations. The initial subprocess mutex
marker reused the panic path; final proof uses a fresh per-mode path and checks
exact contents, so a previous child cannot satisfy the cleanup assertion.

## Phase 5 findings and remaining gates

Browser fetch cannot expose every native transport/header/redirect observation
required by the shared HTTP contract. Basic responses omit forbidden response
headers; CORS responses expose a filtered header set; manual redirects expose
an opaque redirect response with status 0 and no headers/body. These are specified
browser restrictions, not missing wrapper code. See the [Fetch Standard filtered
responses](https://fetch.spec.whatwg.org/#concept-filtered-response) and
[manual redirect behavior](https://fetch.spec.whatwg.org/#concept-request-redirect-mode). Full adapters must resolve that contract
explicitly; they cannot silently weaken it or map this test-only fetch wrapper to
production. Browser WebCrypto key/result ownership needs the registry/decode path
above, cancellation must consume rejected Promises, and cleanup acknowledgement
must include reader/resource release. Browser crypto, production HTTP, package
emission/declarations and SDK/KAS interoperability remain later work. Importable
Promise executable artifacts are not SDK library exports or persistent clients.
No whole Phase 4, browser SDK, all-target host, or library acceptance is claimed.

## Task identity repair after Rust review

Root's copied-runtime probe found that the original operation-only live set allowed
a token for a different task to ACK an existing operation and release its valid
queued completion before real cleanup. The retained failing probe is
`../sdk/.local/root-ts-foreign-ack-review.log`; isolated fixed Node and actual-browser
proofs are recorded separately. Production now uses an operation-to-task map and
checks exact task identity before accepting either publication or ACK. The shared
portable regression rejects foreign result/fault/ACK, retains the canonical result
until its own cleanup ACK, then verifies one cleanup/completion and bounded late
records. Current production verification is retained under
`out/root-ts-task-identity-repair/`; root accepted its terminal0 launcherPID3255438/session23850 with all13
statuses0: Node, actual Chromium147.0.7727.15, typecheck, emitted integration
7.772 seconds, all-seven contracts144.752 seconds,179 harness cases, freshness503,
Rust preservation and whitespace. Exact repaired source/design hashes are in
`out/root-ts-task-identity-repair/accepted-source-freeze.sha256`.
Historical TypeScript evidence above is not relabeled as having covered this case.

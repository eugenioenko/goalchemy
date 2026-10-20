# CI prerequisites and coverage

Linux x86_64 CI uses at most four runners across the repository, including
overlapping PRs, optional tests, documentation builds and deployments. Jobs
share four `repository-runner-*` concurrency groups with `queue: max`, so
waiting jobs queue without occupying runners or replacing one another. Each
group can hold up to 100 pending jobs. These groups enforce the four-runner cap
without a second matrix limit holding up jobs for an available slot.
New commits cancel superseded runs of the same workflow,
including older pushes to `main`. Closing or merging a PR also cancels its
remaining workflows. A `pull_request_target: closed` event shares the PR's
workflow concurrency group, with every job skipped. This cancellation needs
no runner and executes no PR code. Post-merge `push` runs have a separate group,
so tests and documentation publishing on `main` continue.

Every new runner job must use one of these four groups. Workflow-level
cancellation groups stay separate so workflows do not block their own jobs.

The original `go test -short ./...` coverage is partitioned into core packages, native
contracts/browser checks, and four integration shards. Float differential
tests, readable/compact naming tests, and unshort runtime conformance each run
in their own job across all targets. Each target has its own "Language and
corpus (<target>)" job that runs every language fixture and corpus regression
for that target without short mode; the Go job also runs the feature manifest
check. "Examples and reproducible output all targets" runs `TestExamples` and
`TestReproducibleOutput` without short mode, so every example program and the
byte-for-byte reproducibility check cover all eight targets. "Memory and
sanitizers" runs `tests/memory` (Rust and C heap stress) and `tests/sanitize`
(every language fixture and corpus regression on C built with the pinned
clang's address and undefined-behavior sanitizers) without short mode; a
missing pinned clang fails the job rather than skipping it. The short-mode jobs
retain the native integration, actual Chromium, and focused C ASan/UBSan
checks.

`scripts/ci-suite.py` discovers packages with `go list ./...`; packages outside
the explicitly separated suites join the core job automatically. Integration
shards discover top-level tests, fuzz seed tests and examples with `go test
-list .`, sort their names and distribute them across four jobs. Subtests stay
with their parent. The core job audits that the partitions cover the discovered
packages and integration tests without overlap, and that the workflow has
exactly one correctly named language job for every target listed in
`tests/language` and `tests/corpus`. No test-source changes or
new test skip conditions are needed. Setup selects the prerequisites for each
suite without changing its test commands. The final `test` job preserves the existing check
name and succeeds only when every matrix job succeeds; failures do not cancel
the other shards.

`scripts/ci-suite.py <suite> --target <target> --setup` lists each job's
prerequisites before any language toolchains are installed. Language jobs
prepare only their selected target. Native pause jobs prepare their selected
targets; the combined Go/TypeScript/C# job also prewarms the Go library probe.
Mixed core, contracts and integration suites retain full setup. Examples,
floats and naming need all language runtimes but no browser, sanitizer LLVM or
runtime harness prebuild. Memory checks need C, Rust and LLVM. Runtime
conformance prebuilds the Rust harness but does not install a browser.

`scripts/ci-bootstrap.sh --suite <suite> --target <target>` prepares that
selection. Omitting the suite preserves full local setup. Dependency selection
is covered by focused tests, including the shell bootstrap and toolchain cache
validation. CI runs these tests in the core job.

Installed .NET, LLVM and Swift toolchains have separate caches keyed by the
OS, architecture and toolchain lock. Restored installations must have their
expected executable and matching pin marker; a mismatch invokes the existing
hash-verified download and installation path. Jobs cache Python wheel downloads,
locked NuGet packages, Java dependencies, and the shared Cargo dependency and
build directories they actually use. Rust caches include the native harness's
build directory as well as generated fixture builds.

Native C needs a C compiler, pkg-config, Boehm GC, libcurl, OpenSSL and
Perl development tools. Its fault-output assertions also require ripgrep. CI
installs those OS packages and Playwright's browser
OS dependencies. The existing hash-verified `toolchains.lock` fetcher provides
repository-local .NET and LLVM/libtinfo for tests that require those paths.
Java remains the workflow's Temurin 21 installation.

Swift uses the hash-verified Swift 6.4.0 Ubuntu 22.04 Linux x86_64 toolchain
from `toolchains.lock`, with Swift 5 language mode. Its native module requires
OpenSSL 3, zlib and libcurl development files, a C compiler and pkg-config.
CI installs those packages before bootstrap. `driver.ToolEnv()` also discovers
the local Swift binary and optional `.toolchains/pkgconfig` directory; this
allows local development without changing system profiles.

Python uses CPython 3.10 and a dedicated `.toolchains/ci-python` environment.
`ci/python-requirements.txt` pins and hashes binary wheels for the Linux x86_64
release baseline. Browser/build tools use `ci/browser-tooling/package-lock.json`
and are installed at the existing browser fixtures' tooling path under ignored
`out/`. Chromium is the revision supplied by pinned Playwright; bootstrap sets
`GOALCHEMY_CHROME` so fixtures do not select an unrelated system Chrome. Runtime
TypeScript npm dependencies are also installed, so its strict typecheck runs.
These tools are outside the production browser import graph.

The Rust harness copies `Cargo.lock` into its temporary build and runs with
`--locked --offline`. Bootstrap first fetches that locked native dependency
graph, so a clean runner does not depend on a developer's Cargo cache. Cached
downloads are optional; their lock/checksum checks still run.
Suites that run runtime contracts also prebuild the actual harness with empty
input before timed checks begin. This keeps a cold vendored OpenSSL build out
of the CRC harness's startup deadline without changing cases or timeouts.

C# checksum builds use Microsoft's `System.IO.Hashing` 8.0.0 NuGet package.
Bootstrap restores `targets/csharp/checksum.csproj` against its version and
content-hash lock before the full-runtime harness and tests run. A NuGet cache
is optional; locked restore fetches the package on a clean runner. Generated
C# projects add the same locked dependency only when IEEE CRC32 is linked.

The frontend reference Go toolchain is read from its source constant and
prewarmed, alongside Go 1.25.14 used by the native HTTP probe. That probe builds
with the race detector before running the binary, keeping build/download
messages in diagnostics and preserving exact checks on all program output.

For local Linux x86_64 checks, install the same OS prerequisites and run:

```sh
bash scripts/ci-bootstrap.sh # CPython 3.10, Go, Node 22+, Cargo and Java 21 on PATH
source out/ci/environment.sh
go vet ./...
go run ./cmd/goalchemy spec generate -check
go test -short -timeout 30m ./...
go test -v -timeout 15m ./tests/contracts -run '^TestTargetConformance$' -count=1
```

To reproduce a specific CI job, run `python3 scripts/ci-suite.py core`,
`contracts`, `examples`, `memory`, `floats`, `naming`, or `runtime` after matching bootstrap. Language jobs
use `python3 scripts/ci-suite.py language --target <target>`, for example
`--target rust`. Integration jobs use `python3 scripts/ci-suite.py integration
--shard 0` through `--shard 3`. Add `--plan` to inspect commands without executing them,
or use `python3 scripts/ci-suite.py --verify-plan` to audit coverage.

For a Python language job, for example:

```sh
bash scripts/ci-bootstrap.sh --suite language --target python
source out/ci/environment.sh
python3 scripts/ci-suite.py language --target python
```

Use `--with-browser-deps` when a selected browser suite should also install
Chromium's OS prerequisites. CI writes its runtime paths to `GITHUB_PATH`/`GITHUB_ENV`; local
bootstrap writes only ignored `out/ci/environment.sh` and changes no profiles.

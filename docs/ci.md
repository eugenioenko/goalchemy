# CI prerequisites and coverage

The Linux x86_64 CI job keeps the existing `go test -short ./...` suite and adds
an unshort `TestTargetConformance` run over Go, TypeScript, Python, Java, C#,
Rust, and C. Short mode limits the general language/example matrices; it still
runs the native integration, actual Chromium, and focused C ASan/UBSan checks.
The separate contract step has no `-short` flag or target filter. The full
sanitizer/corpus/memory matrices retain their existing short-mode behavior.

`scripts/ci-bootstrap.sh` prepares the prerequisites rather than skipping
checks. Native C needs a C compiler, pkg-config, Boehm GC, libcurl, OpenSSL and
Perl development tools. Its fault-output assertions also require ripgrep. CI
installs those OS packages and Playwright's browser
OS dependencies. The existing hash-verified `toolchains.lock` fetcher provides
repository-local .NET and LLVM/libtinfo for tests that require those paths.
Java remains the workflow's Temurin 21 installation.

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
Bootstrap also builds the actual harness with empty input before timed contract
checks and parallel tests begin. This keeps a cold vendored OpenSSL build out
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

Use `--with-browser-deps` when bootstrap should also install Chromium's OS
prerequisites. CI writes its runtime paths to `GITHUB_PATH`/`GITHUB_ENV`; local
bootstrap writes only ignored `out/ci/environment.sh` and changes no profiles.

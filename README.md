# Goalchemy

Goalchemy transpiles a restricted, Go-compatible language into other languages. Programs are ordinary `.go` files that type-check with Go, and the compiler accepts only the features in [the language specification](specs/language.md). [The plan](plan.md) describes the architecture and milestones, and [docs/usage.md](docs/usage.md) is the user guide.

Release 0.1 compiles the sequential and cooperative language to lowered Go, TypeScript for Node.js, Python 3.10+, Java 21, C# (.NET 8), Rust, and C17 (with the Boehm collector; non-main packages build as C libraries). `examples/` holds runnable programs, each with a `goalchemy.yaml`.

```sh
go build -o bin/goalchemy ./cmd/goalchemy
bin/goalchemy check ./path/to/package                  # validate against the sequential gate
bin/goalchemy compile -target go -out out/ ./path/pkg  # emit a runnable target directory
bin/goalchemy compile -target ir ./path/pkg            # print the typed IR
bin/goalchemy run -target typescript ./examples/calc   # compile and run
(cd examples/bank && ../../bin/goalchemy build)        # build every target in goalchemy.yaml
bin/goalchemy features                                 # supported feature matrix
bin/goalchemy spec validate                            # validate the contract catalog
bin/goalchemy spec generate [-check]                   # regenerate target specs and harness bindings
bin/goalchemy test                                     # run contract cases through target harnesses
go test -timeout 30m ./...                             # full seven-target suite
go test -short -timeout 5m ./...                        # development check
```

## Layout

- `specs/`: language specification, diagnostic codes, JSON Schemas, canonical type and runtime function contracts.
- `internal/frontend`, `internal/subset`: package loading and the language-gate validator.
- `internal/ir`, `internal/lower`: the typed IR and the lowering that decides evaluation order, copies, and receiver adaptation.
- `internal/emit/<target>`: target emitters. `internal/link` copies the runtime files a program needs and writes `goalchemy.manifest.json`.
- `targets/<target>/`: `target.yaml` mapping, one runtime file per contract function, generated `spec/` files, and the conformance harness.
- `tests/language/testdata`: source fixtures compared against native Go; `tests/corpus`: saved regressions with metadata; `examples/`: documented programs.

Diagnostic codes are listed in [specs/diagnostics.md](specs/diagnostics.md).

Run `scripts/fetch-toolchains.sh` to install the pinned Linux x64 JDK, .NET
SDK, bdwgc, and LLVM toolchains before the full test suite.

For seven-target differential campaigns, performance baselines, and toolchain
upgrade reports, see [docs/hardening.md](docs/hardening.md).

Goalchemy is licensed under [Apache-2.0](LICENSE). See [NOTICE](NOTICE).

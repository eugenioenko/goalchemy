# Goalchemy

Goalchemy transpiles a restricted, Go-compatible language into other languages. Programs are ordinary `.go` files that type-check with Go, and the compiler accepts only the features in [the language specification](specs/language.md). [The plan](plan.md) describes the architecture and milestones.

```sh
go build -o bin/goalchemy ./cmd/goalchemy
bin/goalchemy check ./path/to/package                  # validate against the sequential gate
bin/goalchemy compile -target go -out out/ ./path/pkg  # emit a runnable target directory
bin/goalchemy compile -target ir ./path/pkg            # print the typed IR
bin/goalchemy spec validate                            # validate the contract catalog
bin/goalchemy spec generate [-check]                   # regenerate target specs and harness bindings
bin/goalchemy test                                     # run contract cases through target harnesses
go test ./...                                          # unit, contract, language, and integration tests
```

## Layout

- `specs/`: language specification, diagnostic codes, JSON Schemas, canonical type and runtime function contracts.
- `internal/frontend`, `internal/subset`: package loading and the language-gate validator.
- `internal/ir`, `internal/lower`: the typed IR and the lowering that decides evaluation order, copies, and receiver adaptation.
- `internal/emit/<target>`: target emitters. `internal/link` copies the runtime files a program needs and writes `goalchemy.manifest.json`.
- `targets/<target>/`: `target.yaml` mapping, one runtime file per contract function, generated `spec/` files, and the conformance harness.
- `tests/language/testdata`: source fixtures compared against native Go.

Diagnostic codes are listed in [specs/diagnostics.md](specs/diagnostics.md).

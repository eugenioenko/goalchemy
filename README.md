# Goalchemy

Goalchemy transpiles a restricted, Go-compatible language into other languages. Programs are ordinary `.go` files that type-check with Go, and the compiler accepts only the features in [the language specification](specs/language.md). [The plan](plan.md) describes the architecture and milestones.

```sh
go build -o bin/goalchemy ./cmd/goalchemy
bin/goalchemy check ./path/to/package       # validate against the sequential gate
bin/goalchemy spec validate -root .         # validate the contract catalog
go test ./...
```

Diagnostic codes are listed in [specs/diagnostics.md](specs/diagnostics.md).

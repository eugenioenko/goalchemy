# Goalchemy

Goalchemy compiles a [restricted, Go-compatible language](specs/language.md) to seven targets. Source programs are ordinary `.go` files that also run with Go.

Install the compiler with Go 1.27.1:

```sh
go install github.com/eugenioenko/goalchemy/cmd/goalchemy@latest
```

## Quick start

1. Create a Go module and `main.go`:

   ```sh
   mkdir hello && cd hello
   go mod init example.com/hello
   cat > main.go <<'GO'
   package main

   func main() {
       println("Hello from Goalchemy")
   }
   GO
   ```

2. Compile and run it with Python:

   ```sh
   goalchemy run -target python .
   ```

## Target prerequisites

| Target | To run generated programs |
| --- | --- |
| Go | Go 1.25 or later |
| TypeScript | Node.js 22.6 or later |
| Python | Python 3.10 or later |
| Java | JDK 21 or later |
| C# | .NET SDK 8 |
| Rust | Stable Rust toolchain |
| C | C17 compiler and bdwgc 8.x with threads |

The [usage guide](docs/usage.md) covers commands, project configuration, language gates, and output. See [follow-ups](docs/followups.md) for known limits and future work, and [hardening](docs/hardening.md) for test and toolchain setup.

Goalchemy is licensed under [Apache-2.0](LICENSE). See [NOTICE](NOTICE).

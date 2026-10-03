package driver

import (
	"errors"
	"fmt"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	cemit "github.com/eugenioenko/goalchemy/internal/emit/c"
	"github.com/eugenioenko/goalchemy/internal/emit/csharp"
	"github.com/eugenioenko/goalchemy/internal/emit/golang"
	"github.com/eugenioenko/goalchemy/internal/emit/java"
	"github.com/eugenioenko/goalchemy/internal/emit/py"
	"github.com/eugenioenko/goalchemy/internal/emit/rust"
	"github.com/eugenioenko/goalchemy/internal/emit/ts"
	"github.com/eugenioenko/goalchemy/internal/link"
)

func init() {
	Register("go", emitGo)
}

func emitErr(code, msg string) []diagnostics.Diagnostic {
	return []diagnostics.Diagnostic{{Code: code, Severity: diagnostics.Error, Feature: "emission", Message: msg,
		Remedy: "This is a compiler defect; please report it with the source program."}}
}

func emitGo(res *Result, out string) []diagnostics.Diagnostic {
	abs, _ := filepath.Abs(out)
	o, err := golang.Emit(res.IR, abs, symbols(res, "go"))
	if err != nil {
		var boundary *golang.LibraryBoundaryError
		if errors.As(err, &boundary) {
			return []diagnostics.Diagnostic{{Code: "GCE007", Severity: diagnostics.Error, Feature: "Go library boundary", Message: boundary.Error(), Remedy: "Use public struct value trees, primitive arrays/slices and a final error result; native pointers, maps, channels and arbitrary callbacks are outside this library ABI."}}
		}
		if o != nil {
			_ = link.WriteFile(out, "main.go", o.Source)
		}
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "go", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "go", files, out, "rt", true)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	if res.IR.Library {
		for _, name := range []string{"library.go", "lib_callback_request.go"} {
			linked := false
			for _, file := range rtFiles {
				if file == "rt/"+name {
					linked = true
					break
				}
			}
			if linked {
				continue
			}
			src, readErr := fs.ReadFile(res.Catalog.FS, "targets/go/runtime/"+name)
			if readErr != nil {
				return emitErr("GCE005", readErr.Error())
			}
			if err := link.WriteFile(out, "rt/"+name, src); err != nil {
				return emitErr("GCE005", err.Error())
			}
			rtFiles = append(rtFiles, "rt/"+name)
		}
	}
	// Bundle stdlib-only native capabilities in the generated module.
	bundled := map[string]bool{}
	for _, ref := range refs {
		parts := strings.Split(ref.ID, ".")
		if len(parts) != 3 || parts[0] != "lib" || (parts[1] != "crypto" && parts[1] != "encoding" && parts[1] != "clock" && parts[1] != "http") {
			continue
		}
		pkg := parts[1]
		if bundled[pkg] {
			continue
		}
		bundled[pkg] = true
		src, err := fs.ReadFile(res.Catalog.FS, "lib/"+pkg+"/"+pkg+".go")
		if err != nil {
			return emitErr("GCE005", err.Error())
		}
		dest := "cap/" + pkg + "/" + pkg + ".go"
		if err = link.WriteFile(out, dest, src); err != nil {
			return emitErr("GCE005", err.Error())
		}
		rtFiles = append(rtFiles, dest)
	}
	for _, file := range rtFiles {
		if !strings.HasPrefix(file, "rt/") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			return emitErr("GCE005", err.Error())
		}
		for pkg := range bundled {
			src = []byte(strings.ReplaceAll(string(src), "github.com/eugenioenko/goalchemy/lib/"+pkg, "goalchemyout/cap/"+pkg))
		}
		if err = link.WriteFile(out, file, src); err != nil {
			return emitErr("GCE005", err.Error())
		}
	}
	if err := link.WriteFile(out, "main.go", o.Source); err != nil {
		return emitErr("GCE005", err.Error())
	}
	if err := link.WriteFile(out, "go.mod", []byte("module goalchemyout\n\ngo 1.25\n")); err != nil {
		return emitErr("GCE005", err.Error())
	}
	usage := "go run ."
	if res.IR.Library {
		usage = "go build ./... (import module goalchemyout, package generated; replace module path for distribution)"
	}
	if err := link.WriteFile(out, "README.md", []byte(readme("go", usage, "Requires Go 1.25 or later."))); err != nil {
		return emitErr("GCE005", err.Error())
	}
	if err := link.WriteManifest(out, res.Catalog, "go", refs, rtFiles, []string{"README.md", "go.mod", "main.go"}, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

func init() {
	Register("typescript", emitTS)
}

func emitTS(res *Result, out string) []diagnostics.Diagnostic {
	o, err := ts.Emit(res.IR, symbols(res, "typescript"))
	if err != nil {
		var boundary *ts.LibraryBoundaryError
		if errors.As(err, &boundary) {
			return []diagnostics.Diagnostic{{Code: "GCE007", Severity: diagnostics.Error, Feature: "TypeScript library boundary", Message: boundary.Error(), Remedy: "Use bounded public value trees and a final error result."}}
		}
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "typescript", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "typescript", files, out, "rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	if res.IR.Library {
		filtered := files[:0]
		for _, f := range files {
			if f != "types/node_host.ts" {
				filtered = append(filtered, f)
			}
		}
		files = filtered
		kept := rtFiles[:0]
		for _, f := range rtFiles {
			if f != "rt/types/node_host.ts" {
				kept = append(kept, f)
			}
		}
		rtFiles = kept
		_ = os.Remove(filepath.Join(out, "rt/types/node_host.ts"))
		src, readErr := fs.ReadFile(res.Catalog.FS, "targets/typescript/runtime/library.ts")
		if readErr != nil {
			return emitErr("GCE005", readErr.Error())
		}
		if err := link.WriteFile(out, "rt/runtime/library.ts", src); err != nil {
			return emitErr("GCE005", err.Error())
		}
		files = append(files, "runtime/library.ts")
		rtFiles = append(rtFiles, "rt/runtime/library.ts")
	}
	var index strings.Builder
	index.WriteString("// Code generated by goalchemy. DO NOT EDIT.\n")
	for _, f := range files {
		if f == "types/node_host.ts" {
			continue
		}
		fmt.Fprintf(&index, "export * from \"./%s\";\n", f)
	}
	abs, _ := filepath.Abs(out)
	if res.IR.Cooperative {
		o.SourceMap.File = "program.ts"
	}
	smap, err := o.SourceMap.JSON(abs)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	gen := map[string][]byte{
		"main.ts":      o.Source,
		"main.ts.map":  smap,
		"rt/index.ts":  []byte(index.String()),
		"package.json": []byte("{\n  \"type\": \"module\",\n  \"private\": true,\n  \"engines\": {\"node\": \">=22.6\"}\n}\n"),
		"README.md":    []byte(readme("typescript", "node main.ts", "Requires Node.js 22.6 or later (TypeScript type stripping); Node 22 needs --experimental-strip-types.")),
	}
	if res.IR.Library {
		gen["main.ts"] = o.Source
		gen["tsconfig.json"] = []byte(`{"compilerOptions":{"target":"ES2022","module":"NodeNext","moduleResolution":"NodeNext","rewriteRelativeImportExtensions":true,"declaration":true,"outDir":"dist","strict":true,"skipLibCheck":true,"lib":["ES2022","DOM","DOM.Iterable"]},"include":["main.ts","rt/**/*.ts"]}`)
		gen["package.json"] = []byte(`{"name":"goalchemy-generated","version":"0.0.0","type":"module","private":true,"exports":{".":{"types":"./dist/main.d.ts","import":"./dist/main.js"}},"files":["dist"],"engines":{"node":">=22.6"},"scripts":{"build":"tsc -p ."}}`)
		gen["README.md"] = []byte(readme("typescript", "tsc -p . (TypeScript >=5.7); import ./dist/main.js", "Portable Node/browser Promise library; Uint8Array bytes and bigint int64."))
	} else if res.IR.Cooperative {
		gen["program.ts"] = []byte(strings.ReplaceAll(string(o.Source), "sourceMappingURL=main.ts.map", "sourceMappingURL=program.ts.map"))
		gen["program.ts.map"] = smap
		delete(gen, "main.ts.map")
		gen["main.ts"] = []byte("// Node executable entry.\nimport \"./rt/types/node_host.ts\";\nimport { $run } from \"./program.ts\";\n$run();\n")
		gen["host.ts"] = []byte("// Portable Promise entry; executable globals are serialized, not library instances.\nexport { $runHost as runHost } from \"./program.ts\";\n")
	}
	var names []string
	for name, data := range gen {
		if err := link.WriteFile(out, name, data); err != nil {
			return emitErr("GCE005", err.Error())
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if err := link.WriteManifest(out, res.Catalog, "typescript", refs, rtFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

func readme(target, run, req string) string {
	return "# Goalchemy " + target + " output\n\nGenerated by goalchemy; do not edit. Run with:\n\n```sh\n" + run + "\n```\n\n" + req +
		" The runtime files the program needs are under rt/, and goalchemy.manifest.json lists the contracts, versions, and files.\n"
}

// symbols maps capability contracts to the target's runtime symbols.
func symbols(res *Result, target string) map[string]string {
	out := map[string]string{}
	if t, ok := res.Catalog.Targets[target]; ok {
		for _, f := range t.Functions {
			out[f.ID] = f.Symbol
		}
	}
	return out
}

func init() {
	Register("python", emitPython)
}

func emitPython(res *Result, out string) []diagnostics.Diagnostic {
	o, err := py.Emit(res.IR, symbols(res, "python"))
	if err != nil {
		if _, ok := err.(*py.LibraryBoundaryError); ok {
			return emitErr("GCE007", err.Error())
		}
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "python", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "python", files, out, "rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	var index strings.Builder
	index.WriteString("# Code generated by goalchemy. DO NOT EDIT.\n# flake8: noqa\n")
	for _, f := range files {
		mod := strings.TrimSuffix(strings.ReplaceAll(f, "/", "."), ".py")
		fmt.Fprintf(&index, "from .%s import *\n", mod)
	}
	abs, _ := filepath.Abs(out)
	var lines strings.Builder
	keys := make([]int, 0, len(o.Lines))
	for k := range o.Lines {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		pos := o.Lines[k]
		name := pos.Filename
		if rel, err := filepath.Rel(abs, name); err == nil {
			name = filepath.ToSlash(rel)
		}
		fmt.Fprintf(&lines, "%d\t%s:%d:%d\n", k, name, pos.Line, pos.Column)
	}
	gen := map[string][]byte{
		"main.py":                o.Source,
		"main.py.lines":          []byte(lines.String()),
		"rt/__init__.py":         []byte(index.String()),
		"rt/types/__init__.py":   []byte(""),
		"rt/runtime/__init__.py": []byte(""),
		"README.md":              []byte(readme("python", "python3 main.py", "Requires Python 3.10 or later.")),
	}
	if res.IR.Library {
		gen["__init__.py"] = []byte("from .main import *\n")
	}
	var names []string
	for name, data := range gen {
		if err := link.WriteFile(out, name, data); err != nil {
			return emitErr("GCE005", err.Error())
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if err := link.WriteManifest(out, res.Catalog, "python", refs, rtFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

func init() {
	Register("java", emitJava)
}

const javaRun = `#!/bin/sh
# Compiles and runs the program with a Java 21 or later JDK.
set -e
cd "$(dirname "$0")"
if [ -n "$JAVA_HOME" ]; then PATH="$JAVA_HOME/bin:$PATH"; fi
javac -nowarn -encoding UTF-8 -d classes Main.java rt/types/*.java rt/runtime/*.java
exec java -cp classes Main
`

const javaLibraryBuild = `#!/bin/sh
set -eu
cd "$(dirname "$0")"
if [ -n "${JAVA_HOME:-}" ]; then PATH="$JAVA_HOME/bin:$PATH"; fi
mkdir -p classes
javac -nowarn -encoding UTF-8 -d classes Generated.java rt/types/*.java rt/runtime/*.java
jar --create --date=2026-01-01T00:00:00Z --file goalchemy-generated.jar -C classes .
`

func emitJava(res *Result, out string) []diagnostics.Diagnostic {
	o, err := java.Emit(res.IR, symbols(res, "java"))
	if err != nil {
		var boundary *java.LibraryBoundaryError
		if errors.As(err, &boundary) {
			return []diagnostics.Diagnostic{{Code: "GCE007", Severity: diagnostics.Error, Feature: "Java library boundary", Message: boundary.Error(), Remedy: "Use bounded public value trees and a final error result."}}
		}
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "java", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "java", files, out, "rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	if res.IR.Library {
		for _, name := range rtFiles {
			data, err := os.ReadFile(filepath.Join(out, name))
			if err != nil {
				return emitErr("GCE005", err.Error())
			}
			data = []byte(strings.ReplaceAll(string(data), "package rt;", "package io.goalchemy.runtime;"))
			if err := link.WriteFile(out, name, data); err != nil {
				return emitErr("GCE005", err.Error())
			}
		}
		source := []byte(strings.ReplaceAll(string(o.Source), "import rt.*;", "import io.goalchemy.runtime.*;"))
		gen := map[string][]byte{"Generated.java": source, "Generated.java.lines": lineTable(o.Lines, out), "build.sh": []byte(javaLibraryBuild), "README.md": []byte(readme("java", "sh build.sh; import io.goalchemy.generated.Generated from goalchemy-generated.jar", "JDK21; serialized cancellable value operations; production crypto additionally requires declared BC1.86."))}
		var names []string
		for name, data := range gen {
			if err := link.WriteFile(out, name, data); err != nil {
				return emitErr("GCE005", err.Error())
			}
			names = append(names, name)
		}
		sort.Strings(names)
		if err := link.WriteManifest(out, res.Catalog, "java", refs, rtFiles, names, res.Program); err != nil {
			return emitErr("GCE005", err.Error())
		}
		return nil
	}
	gen := map[string][]byte{
		"Main.java":       o.Source,
		"Main.java.lines": lineTable(o.Lines, out),
		"run.sh":          []byte(javaRun),
		"README.md":       []byte(readme("java", "sh run.sh", "Requires a Java 21 or later JDK.")),
	}
	var names []string
	for name, data := range gen {
		if err := link.WriteFile(out, name, data); err != nil {
			return emitErr("GCE005", err.Error())
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if err := link.WriteManifest(out, res.Catalog, "java", refs, rtFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

// lineTable renders a generated-line to source-position table.
func lineTable(m map[int]token.Position, out string) []byte {
	abs, _ := filepath.Abs(out)
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		pos := m[k]
		name := pos.Filename
		if rel, err := filepath.Rel(abs, name); err == nil {
			name = filepath.ToSlash(rel)
		}
		fmt.Fprintf(&b, "%d\t%s:%d:%d\n", k, name, pos.Line, pos.Column)
	}
	return []byte(b.String())
}

func init() {
	Register("csharp", emitCSharp)
}

const csharpRun = `#!/bin/sh
# Compiles and runs the program with the .NET 8 SDK's C# compiler.
set -e
cd "$(dirname "$0")"
dotnet=dotnet
if [ -n "$DOTNET_ROOT" ] && [ -x "$DOTNET_ROOT/dotnet" ]; then dotnet="$DOTNET_ROOT/dotnet"; fi
root=$(dirname "$(readlink -f "$(command -v "$dotnet")")")
sdk=$("$dotnet" --list-sdks | awk '/^8\./ {v=$1} END {print v}')
csc="$root/sdk/$sdk/Roslyn/bincore/csc.dll"
ref=$(ls -d "$root"/packs/Microsoft.NETCore.App.Ref/8.*/ref/net8.0 | tail -n 1)
mkdir -p bin
refs=""
for f in "$ref"/*.dll; do refs="$refs -r:$f"; done
"$dotnet" "$csc" -nologo -noconfig -nostdlib -nowarn:CS0162,CS0164,CS0168,CS0219,CS8981 -langversion:12 -nullable:disable \
  -optimize+ -out:bin/main.dll $refs Main.cs rt/types/*.cs rt/runtime/*.cs >&2
cat > bin/main.runtimeconfig.json <<'JSON'
{"runtimeOptions": {"tfm": "net8.0", "framework": {"name": "Microsoft.NETCore.App", "version": "8.0.0"}}}
JSON
exec "$dotnet" bin/main.dll
`

const csharpProject = `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>disable</Nullable>
    <ImplicitUsings>disable</ImplicitUsings>
    <NoWarn>CS0162;CS0164;CS0168;CS0219;CS8981</NoWarn>
  </PropertyGroup>
</Project>
`

func emitCSharp(res *Result, out string) []diagnostics.Diagnostic {
	o, err := csharp.Emit(res.IR, symbols(res, "csharp"))
	if err != nil {
		if _, ok := err.(*csharp.LibraryBoundaryError); ok {
			return emitErr("GCE007", err.Error())
		}
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "csharp", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "csharp", files, out, "rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	project := csharpProject
	if res.IR.Library {
		project = strings.Replace(project, "<OutputType>Exe</OutputType>", "<OutputType>Library</OutputType>", 1)
	}
	gen := map[string][]byte{
		"Main.cs":       o.Source,
		"Main.cs.lines": lineTable(o.Lines, out),
		"main.csproj":   []byte(project),
		"run.sh":        []byte(csharpRun),
		"README.md":     []byte(readme("csharp", "sh run.sh", "Requires the .NET 8 SDK; dotnet run also works with main.csproj.")),
	}
	if res.IR.Library {
		delete(gen, "run.sh")
		gen["build.sh"] = []byte("#!/bin/sh\nset -eu\ndotnet build main.csproj -c Release -o lib --nologo\n")
		gen["README.md"] = []byte(readme("csharp", "sh build.sh; reference lib/main.dll and Goalchemy.Generated.GoProgram", ".NET8; serialized cancellable copied value operations, byte-preserving generic strings and typed Rt.Library.Failure."))
	}
	var names []string
	for name, data := range gen {
		if err := link.WriteFile(out, name, data); err != nil {
			return emitErr("GCE005", err.Error())
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if err := link.WriteManifest(out, res.Catalog, "csharp", refs, rtFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

func init() {
	Register("rust", emitRust)
}

const rustRun = `#!/bin/sh
# Compiles and runs the program with a stable Rust toolchain.
set -e
cd "$(dirname "$0")"
rustc --edition 2021 -C opt-level=2 -C debuginfo=0 -o main src/main.rs >&2
exec ./main
`

const rustCargo = `[package]
name = "goalchemy-out"
version = "0.1.0"
edition = "2021"

[[bin]]
name = "main"
path = "src/main.rs"

[profile.release]
opt-level = 2
`

func emitRust(res *Result, out string) []diagnostics.Diagnostic {
	o, err := rust.Emit(res.IR, symbols(res, "rust"))
	if err != nil {
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "rust", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "rust", files, out, "src/rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	var mod strings.Builder
	mod.WriteString("// Code generated by goalchemy. DO NOT EDIT.\n")
	for _, f := range files {
		name := strings.TrimSuffix(path.Base(f), ".rs")
		fmt.Fprintf(&mod, "#[path = %q]\npub mod %s;\npub use %s::*;\n", f, name, name)
	}
	gen := map[string][]byte{
		"src/main.rs":       o.Source,
		"src/main.rs.lines": lineTable(o.Lines, out),
		"src/rt/mod.rs":     []byte(mod.String()),
		"Cargo.toml":        []byte(rustCargo),
		"run.sh":            []byte(rustRun),
		"README.md":         []byte(readme("rust", "sh run.sh", "Requires a stable Rust toolchain (edition 2021); cargo run --release also works.")),
	}
	var names []string
	for name, data := range gen {
		if err := link.WriteFile(out, name, data); err != nil {
			return emitErr("GCE005", err.Error())
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if err := link.WriteManifest(out, res.Catalog, "rust", refs, rtFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

func init() {
	Register("c", emitC)
}

const cRun = `#!/bin/sh
# Compiles and runs the program with a C17 compiler and the Boehm collector.
# GOALCHEMY_BDWGC may name a bdwgc install prefix; otherwise pkg-config
# bdw-gc or -lgc is used. CC and CFLAGS are honored.
set -e
cd "$(dirname "$0")"
if [ -n "$GOALCHEMY_BDWGC" ]; then
  gc="-I$GOALCHEMY_BDWGC/include $GOALCHEMY_BDWGC/lib/libgc.a"
elif pkg-config --exists bdw-gc 2>/dev/null; then
  gc=$(pkg-config --cflags --libs bdw-gc)
else
  gc=-lgc
fi
${CC:-cc} -std=c17 ${CFLAGS:--O2} -w -Irt/types -o main main.c rt/types/*.c rt/runtime/*.c $gc -lpthread >&2
exec ./main
`

const cLibBuild = `#!/bin/sh
# Builds libgoalchemy.a; link it with bdwgc (-lgc, or GOALCHEMY_BDWGC's
# libgc.a) and -lpthread, and include goalchemy.h.
set -e
cd "$(dirname "$0")"
inc=""
if [ -n "$GOALCHEMY_BDWGC" ]; then inc="-I$GOALCHEMY_BDWGC/include"
elif pkg-config --exists bdw-gc 2>/dev/null; then inc=$(pkg-config --cflags bdw-gc); fi
mkdir -p obj
for f in main.c rt/types/*.c rt/runtime/*.c; do
  ${CC:-cc} -std=c17 ${CFLAGS:--O2} -w -Irt/types $inc -c "$f" -o "obj/$(echo "$f" | tr / _).o"
done
ar rcs libgoalchemy.a obj/*.o
`

func emitC(res *Result, out string) []diagnostics.Diagnostic {
	o, err := cemit.Emit(res.IR, symbols(res, "c"))
	if err != nil {
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "c", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "c", files, out, "rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	gen := map[string][]byte{
		"main.c":       o.Source,
		"main.c.lines": lineTable(o.Lines, out),
		"README.md":    []byte(readme("c", "sh run.sh", "Requires a C17 compiler and the Boehm-Demers-Weiser collector (bdwgc 8.x with threads).")),
	}
	if o.Header != nil {
		gen["goalchemy.h"] = o.Header
		gen["build.sh"] = []byte(cLibBuild)
		gen["README.md"] = []byte(readme("c", "sh build.sh", "Builds libgoalchemy.a; include goalchemy.h and link with bdwgc (-lgc) and -lpthread. Requires a C17 compiler and bdwgc 8.x with threads."))
	} else {
		gen["run.sh"] = []byte(cRun)
	}
	var names []string
	for name, data := range gen {
		if err := link.WriteFile(out, name, data); err != nil {
			return emitErr("GCE005", err.Error())
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if err := link.WriteManifest(out, res.Catalog, "c", refs, rtFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

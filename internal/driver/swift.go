package driver

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	swiftemit "github.com/eugenioenko/goalchemy/internal/emit/swift"
	"github.com/eugenioenko/goalchemy/internal/link"
)

func init() { Register("swift", emitSwift) }

const swiftRun = `#!/bin/sh
set -eu
cd "$(dirname "$0")"
mkdir -p .build/native
${CC:-cc} -O2 -Wno-deprecated-declarations -c rt/native/GoalchemyNative.c -o .build/native/native.o $(pkg-config --cflags openssl zlib libcurl)
build_swift() {
    set -- main.swift
    for source in rt/types/*.swift rt/runtime/*.swift; do
        if [ -f "$source" ]; then set -- "$@" "$source"; fi
    done
    ${SWIFTC:-swiftc} -swift-version 5 -suppress-warnings -I rt/native "$@" .build/native/native.o $(pkg-config --libs openssl zlib libcurl) -o .build/program
}
build_swift
exec .build/program "$@"
`

const swiftLibraryBuild = `#!/bin/sh
set -eu
cd "$(dirname "$0")"
exec swift build -c release "$@"
`

const swiftPackage = `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "GoalchemyGenerated",
    products: [.library(name: "GoalchemyGenerated", targets: ["GoalchemyGenerated"])],
    targets: [
        .systemLibrary(name: "COpenSSL", path: "system/openssl", pkgConfig: "openssl",
                       providers: [.apt(["libssl-dev"]), .brew(["openssl@3"])]),
        .systemLibrary(name: "CZlib", path: "system/zlib", pkgConfig: "zlib",
                       providers: [.apt(["zlib1g-dev"]), .brew(["zlib"])]),
        .systemLibrary(name: "CCurl", path: "system/curl", pkgConfig: "libcurl",
                       providers: [.apt(["libcurl4-openssl-dev"]), .brew(["curl"])]),
        .target(name: "GoalchemyNative", dependencies: ["COpenSSL", "CZlib", "CCurl"],
                path: "rt/native", publicHeadersPath: ".",
                linkerSettings: [.linkedLibrary("crypto"), .linkedLibrary("z"), .linkedLibrary("curl")]),
        .target(name: "GoalchemyGenerated", dependencies: ["GoalchemyNative"], path: ".",
                exclude: ["rt/native", "system", "README.md", "LICENSE", "build.sh", "Generated.swift.lines", "goalchemy.manifest.json"],
                sources: ["Generated.swift", "rt/types", "rt/runtime"])
    ],
    swiftLanguageModes: [.v5]
)
`

func emitSwift(res *Result, out string) []diagnostics.Diagnostic {
	o, err := swiftemit.Emit(res.IR, symbols(res, "swift"))
	if err != nil {
		var boundary *swiftemit.LibraryBoundaryError
		if errors.As(err, &boundary) {
			return emitErr("GCE007", err.Error())
		}
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "swift", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	runtimeFiles, err := link.CopyRuntime(res.Catalog, "swift", files, out, "rt", false)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	sourceName := "main.swift"
	generated := map[string][]byte{
		"run.sh":    []byte(swiftRun),
		"README.md": []byte(readme("swift", "sh run.sh", "Requires Swift 6.4 or later, a C compiler, pkg-config, OpenSSL 3, zlib, and libcurl development headers/libraries. Native dependencies are currently included in every Swift output. The verified baseline is Linux x86_64; Apple SDK integration is not yet verified.")),
	}
	if res.IR.Library {
		sourceName = "Generated.swift"
		delete(generated, "run.sh")
		generated["build.sh"] = []byte(swiftLibraryBuild)
		packageSource := swiftPackage
		hasOperations := false
		for _, file := range runtimeFiles {
			if strings.HasPrefix(file, "rt/runtime/") && strings.HasSuffix(file, ".swift") {
				hasOperations = true
				break
			}
		}
		if !hasOperations {
			packageSource = strings.Replace(packageSource, ", \"rt/runtime\"", "", 1)
		}
		generated["Package.swift"] = []byte(packageSource)
		generated["system/openssl/module.modulemap"] = []byte("module COpenSSL [system] {\n    header \"shim.h\"\n    link \"crypto\"\n    export *\n}\n")
		generated["system/openssl/shim.h"] = []byte("#include <openssl/evp.h>\n")
		generated["system/zlib/module.modulemap"] = []byte("module CZlib [system] {\n    header \"shim.h\"\n    link \"z\"\n    export *\n}\n")
		generated["system/zlib/shim.h"] = []byte("#include <zlib.h>\n")
		generated["system/curl/module.modulemap"] = []byte("module CCurl [system] {\n    header \"shim.h\"\n    link \"curl\"\n    export *\n}\n")
		generated["system/curl/shim.h"] = []byte("#include <curl/curl.h>\n")
		generated["README.md"] = []byte(readme("swift", "sh build.sh", "SwiftPM library product/module GoalchemyGenerated. Add this directory as a local package dependency or publish it under your SDK package name. Requires Swift 6.4 or later, OpenSSL 3, zlib, and libcurl development files. Native dependency discovery uses pkg-config. Verified on Linux x86_64; Apple SDK integration is not yet verified."))
	}
	generated[sourceName] = o.Source
	license, err := fs.ReadFile(res.Catalog.FS, "LICENSE")
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	generated["LICENSE"] = license
	keys := make([]int, 0, len(o.Lines))
	for line := range o.Lines {
		keys = append(keys, line)
	}
	sort.Ints(keys)
	abs, err := filepath.Abs(out)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	var positions strings.Builder
	for _, line := range keys {
		pos := o.Lines[line]
		filename := pos.Filename
		if relative, err := filepath.Rel(abs, filename); err == nil {
			filename = filepath.ToSlash(relative)
		}
		fmt.Fprintf(&positions, "%d\t%s:%d:%d\n", line, filename, pos.Line, pos.Column)
	}
	generated[sourceName+".lines"] = []byte(positions.String())
	names := make([]string, 0, len(generated))
	for name := range generated {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := link.WriteFile(out, name, generated[name]); err != nil {
			return emitErr("GCE005", err.Error())
		}
	}
	if err := link.WriteManifest(out, res.Catalog, "swift", refs, runtimeFiles, names, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}

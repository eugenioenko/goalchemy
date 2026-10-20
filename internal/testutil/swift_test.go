package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSwiftSupportCache(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux fixture object reuse")
	}
	root := t.TempDir()
	t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", filepath.Join(root, "cache"))
	log := filepath.Join(root, "commands")
	t.Setenv("SWIFT_TEST_LOG", log)
	compiler := filepath.Join(root, "compiler")
	if err := os.WriteFile(compiler, []byte(`#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$SWIFT_TEST_LOG"
while [ "$#" -gt 0 ]; do
  if [ "$1" = -o ]; then shift; printf object > "$1"; fi
  shift
done
`), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CC", compiler)
	t.Setenv("SWIFTC", compiler)
	build := func(source, types, native string) {
		t.Helper()
		out := t.TempDir()
		for name, data := range map[string]string{
			"goalchemy.manifest.json": `{"generated_files":["main.swift"],"runtime_files":["rt/types/Value.swift","rt/runtime/core_print.swift"]}`,
			"main.swift":              source, "rt/types/Value.swift": types,
			"rt/runtime/core_print.swift": "func printValue() {}",
			"rt/native/GoalchemyNative.c": native,
		} {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(out, name)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out, name), []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := BuildSwift(out); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(out, ".build/program")); err != nil {
			t.Fatal(err)
		}
	}
	build("first program", "types", "native")
	build("different program", "types", "native")
	build("third program", "changed types", "native")
	build("fourth program", "changed types", "changed native")
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	commands := string(data)
	if strings.Count(commands, "-emit-object") != 3 || strings.Count(commands, "-c rt/native/GoalchemyNative.c") != 3 {
		t.Fatalf("support was not reused or invalidated:\n%s", commands)
	}
	if strings.Count(commands, "-frontend -c") != 4 || strings.Count(commands, "-o .build/program") != 4 {
		t.Fatalf("program compilation or linking was skipped:\n%s", commands)
	}
}

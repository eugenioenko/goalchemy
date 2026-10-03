package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// Patch the real builtin before importing compiler output, so correct CRC values
// alone cannot conceal an unused native adapter or a package export mistake.
const nodeCRCSpy = `import assert from 'node:assert/strict';
import zlib from 'node:zlib';
import {syncBuiltinESMExports} from 'node:module';
const original=zlib.crc32,calls=[];
let sentinel=false;
zlib.crc32=bytes=>{assert.ok(bytes instanceof Uint8Array);calls.push(Array.from(bytes));return sentinel?0xfedcba98:original(bytes)};
syncBuiltinESMExports();
`

func TestGeneratedTypeScriptChecksumEntries(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	run := func(t *testing.T, dir, name string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir, cmd.Env = dir, driver.ToolEnv()
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, data)
		}
		return string(data)
	}
	write := func(t *testing.T, dir, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	mod := fmt.Sprintf("module crcprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	for _, gate := range []string{"", "cooperative"} {
		t.Run("executable-"+gate, func(t *testing.T) {
			source, out := t.TempDir(), t.TempDir()
			write(t, source, "go.mod", mod)
			write(t, source, "main.go", `package main
import "github.com/eugenioenko/goalchemy/lib/checksum"
func main(){
 var missing []byte
 println(checksum.CRC32IEEE(missing),checksum.CRC32IEEE([]byte{}))
 bytes:=[]byte("x123456789y")
 println(checksum.CRC32IEEE(bytes[1:10]))
 if string(bytes)!="x123456789y"{panic("input mutated")}
}`)
			if ds := testutil.CompileGate(source, "typescript", out, gate); len(ds) > 0 {
				t.Fatal(ds)
			}
			write(t, out, "consumer.mjs", nodeCRCSpy+`await import('./main.ts');
assert.deepEqual(calls,[[],[],[49,50,51,52,53,54,55,56,57]]);
console.log('PASS generated executable native CRC');`)
			got := run(t, out, "node", "consumer.mjs")
			if !strings.Contains(got, "0 0\n3421780262\nPASS generated executable native CRC") {
				t.Fatal(got)
			}
		})
	}
	t.Run("library-package", func(t *testing.T) {
		source, out := t.TempDir(), t.TempDir()
		write(t, source, "go.mod", mod)
		write(t, source, "library.go", `package crcprobe
import("github.com/eugenioenko/goalchemy/lib/checksum";"github.com/eugenioenko/goalchemy/lib/context")
func CRC32IEEE(ctx context.Context,data []byte)(uint32,error){return checksum.CRC32IEEE(data),nil}
func SliceCRC32IEEE(ctx context.Context,data []byte)(uint32,error){return checksum.CRC32IEEE(data[1:10]),nil}
`)
		if ds := testutil.CompileGate(source, "typescript", out, "cooperative"); len(ds) > 0 {
			t.Fatal(ds)
		}
		// Use the emitted build configuration unchanged, without @types/node.
		run(t, out, "tsc", "-p", ".")
		write(t, out, "consumer.mjs", nodeCRCSpy+`const {portableHost,runtimeHost}=await import('./dist/rt/types/host.js');
const g=await import('goalchemy-generated');
assert.equal(runtimeHost,portableHost,'Node library entry must not install executable I/O/exit');
assert.equal(await g.CRC32IEEE(null),0);
assert.equal(await g.CRC32IEEE(new Uint8Array()),0);
const backing=new Uint8Array([0,120,49,50,51,52,53,54,55,56,57,121,255]);
const input=backing.subarray(1,12),before=backing.slice();
assert.equal(await g.SliceCRC32IEEE(input),0xcbf43926);
assert.deepEqual(backing,before);
assert.deepEqual(calls,[[],[],[49,50,51,52,53,54,55,56,57]]);
sentinel=true;
assert.equal(await g.CRC32IEEE(null),0xfedcba98,'package import must delegate to real builtin');
assert.equal(await (await import('./dist/node.js')).CRC32IEEE(null),0xfedcba98);
console.log('PASS generated Node package/direct entry native CRC');`)
		t.Log(run(t, out, "node", "consumer.mjs"))
		// The browser bundles the package name, exercising its browser/default
		// exports as well as retaining all existing portable graph assertions.
		t.Log(run(t, root, "node", "targets/typescript/tests/checksum_browser.mjs", out))
	})
}

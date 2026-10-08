package swift_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/driver"
	swiftemit "github.com/eugenioenko/goalchemy/internal/emit/swift"
	"github.com/eugenioenko/goalchemy/internal/subset"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func buildSwift(t *testing.T, source string) *driver.Result {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, text := range map[string]string{"go.mod": fmt.Sprintf("module swiftprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %q\n", root), "probe.go": source} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	result, ds := driver.Build(context.Background(), driver.Options{Dir: dir, Gate: subset.Cooperative})
	if diagnostics.HasErrors(ds) {
		t.Fatal(ds)
	}
	return result
}

func TestNamingModes(t *testing.T) {
	result := buildSwift(t, `package main
 import "github.com/eugenioenko/goalchemy/lib/checksum"
 type State struct { value, clone, café int }
 var counter = 2
 func (s State) Amount() int { return s.value+s.clone+s.café }
 func calculate(input int) int {
   class := State{value:input,clone:3,café:5};address:=&class.value;*address+=counter
   fn:=func()int{class:=1;return class};var reader interface{Amount()int}=class;bound:=class.Amount
   queue:=make(chan int,1);queue<-reader.Amount()+bound()+fn();return <-queue
 }
 func main(){println(calculate(1),checksum.CRC32IEEE([]byte("123456789")))}
 `)
	symbols := make([]string, len(result.IR.Funcs))
	for i, f := range result.IR.Funcs {
		symbols[i] = f.Sym
	}
	for _, compact := range []bool{false, true} {
		t.Run(fmt.Sprintf("compact=%v", compact), func(t *testing.T) {
			result.IR.CompactNames = compact
			first, err := swiftemit.Emit(result.IR, map[string]string{"lib.checksum.crc32_ieee": "Custom.crc32"})
			if err != nil {
				t.Fatal(err)
			}
			second, err := swiftemit.Emit(result.IR, map[string]string{"lib.checksum.crc32_ieee": "Custom.crc32"})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first.Source, second.Source) {
				t.Fatal("nondeterministic emission")
			}
			if !strings.Contains(string(first.Source), "Custom.crc32(") {
				t.Fatal("declared runtime symbol ignored")
			}
			if !strings.Contains(string(first.Source), "task.owner.safepoint()") {
				t.Fatal("source frames lack loop collection safepoints")
			}
			if len(first.Lines) == 0 {
				t.Fatal("missing source position map")
			}
			for i, f := range result.IR.Funcs {
				if f.Sym != symbols[i] {
					t.Fatal("emission mutated IR function identity")
				}
			}
			source := string(first.Source)
			if !compact && (!strings.Contains(source, "v_input") || !strings.Contains(source, "_calculate_")) {
				t.Fatal("readable names absent")
			}
			if compact && (strings.Contains(source, "v_input") || strings.Contains(source, "_calculate_")) {
				t.Fatal("compact names ignored")
			}
			dir := t.TempDir()
			if ds := driver.EmitWithOptions("swift", result, dir, driver.EmitOptions{CompactNames: compact}); diagnostics.HasErrors(ds) {
				t.Fatal(ds)
			}
			got, err := testutil.Runners["swift"](dir)
			if err != nil {
				t.Fatal(err)
			}
			if got.Exit != 0 || got.Stderr != "23 3421780262\n" || got.Stdout != "" {
				t.Fatalf("source behavior changed: %s", got)
			}
		})
	}
}

func TestLibraryBoundaryDiagnostics(t *testing.T) {
	for _, compact := range []bool{false, true} {
		for _, name := range []string{"String", "Array", "Result", "Data", "Task", "GoalchemyKey", "GBuffer", "GCryptoKey", "DispatchTime"} {
			t.Run(fmt.Sprintf("compact=%v/%s", compact, name), func(t *testing.T) {
				result := buildSwift(t, "package boundary\ntype "+name+" struct { Value int };func Export(v "+name+")("+name+",error){return v,nil}")
				result.IR.CompactNames = compact
				_, err := swiftemit.Emit(result.IR, nil)
				if _, ok := err.(*swiftemit.LibraryBoundaryError); !ok {
					t.Fatalf("reserved name must fail boundary validation: %v", err)
				}
				ds := driver.EmitWithOptions("swift", result, t.TempDir(), driver.EmitOptions{CompactNames: compact})
				found := false
				for _, d := range ds {
					found = found || d.Code == "GCE007"
				}
				if !found {
					t.Fatalf("missing GCE007: %v", ds)
				}
			})
		}
	}
}

func TestBulkByteEmission(t *testing.T) {
	result := buildSwift(t, `package bytes
 type Octet uint8
 func Join(a,b []Octet) []Octet { return append(a,b...) }
 func Fixed(a [3]Octet) [3]Octet { return a }
 `)
	for _, compact := range []bool{false, true} {
		t.Run(fmt.Sprintf("compact=%v", compact), func(t *testing.T) {
			result.IR.CompactNames = compact
			output, err := swiftemit.Emit(result.IR, nil)
			if err != nil {
				t.Fatal(err)
			}
			source := string(output.Source)
			if strings.Contains(source, "GSlice.append(") && strings.Contains(source, "try GElements(") {
				t.Fatal("bulk byte append or public boundary expands bytes to GValue elements")
			}
			if !strings.Contains(source, "GBuffer(bytes: v, elem:") || !strings.Contains(source, "GSlice.byteSource(v)") {
				t.Fatal("byte public conversion lacks native buffer path")
			}
		})
	}
}

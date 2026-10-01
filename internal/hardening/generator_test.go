package hardening

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

func TestGenerateAndReduce(t *testing.T) {
	a, b := Generate(42, 80), Generate(42, 80)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("seed did not reproduce operations")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", a.Source(), 0); err != nil {
		t.Fatal(err)
	}
	want := a.Operations[30]
	got, checks := Reduce(a, func(p Program) bool {
		for _, op := range p.Operations {
			if op == want {
				return true
			}
		}
		return false
	}, 1000)
	if checks == 0 || len(got.Operations) != 1 || got.Operations[0] != want {
		t.Fatalf("reducer kept %d operations after %d checks", len(got.Operations), checks)
	}
}

func TestGeneratedProgramsTypeCheck(t *testing.T) {
	for seed := uint64(0); seed < 128; seed++ {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "main.go", Generate(seed, 40).Source(), 0)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if _, err := (&types.Config{}).Check("generated", fset, []*ast.File{file}, nil); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
	}
}

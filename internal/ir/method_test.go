package ir

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestMethodIDSeparatesSignatures(t *testing.T) {
	src := `package p

type text = string

type one struct{}

func (one) Error() text      { return "" }
func (one) Unwrap() error    { return nil }
func (one) Read() int        { return 0 }
func (one) Close() error     { return nil }

type two struct{}

func (two) Error(int) string { return "" }
func (two) Unwrap() []error  { return nil }
func (two) Read() string     { return "" }
func (two) Close() error     { return nil }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("p", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	methods := func(name string) map[string]*types.Func {
		out := map[string]*types.Func{}
		ms := types.NewMethodSet(pkg.Scope().Lookup(name).Type())
		for i := 0; i < ms.Len(); i++ {
			fn := ms.At(i).Obj().(*types.Func)
			out[fn.Name()] = fn
		}
		return out
	}
	one, two := methods("one"), methods("two")
	ts := NewTypes()
	for _, m := range []map[string]*types.Func{two, one} {
		for _, fn := range m {
			ts.RecordMethod(fn, false)
		}
	}
	for name, want := range map[string]bool{"Error": false, "Unwrap": false, "Read": true, "Close": false} {
		if got := ts.MethodID(one[name]); strings.Contains(got, "#") != want {
			t.Errorf("one.%s = %q", name, got)
		}
		if got := ts.MethodID(two[name]); strings.Contains(got, "#") != (name != "Close") {
			t.Errorf("two.%s = %q", name, got)
		}
	}
	if ts.MethodID(one["Read"]) == ts.MethodID(two["Read"]) {
		t.Error("Read signatures share an identity")
	}
	if NewTypes().MethodID(two["Read"]) != "Read" {
		t.Error("unrecorded methods keep their name")
	}
}

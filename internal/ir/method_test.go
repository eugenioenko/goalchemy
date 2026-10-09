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

func TestMethodIDSeparatesWellKnownSignatures(t *testing.T) {
	src := `package p

type text = string

type one struct{}

func (one) Unwrap() error { return nil }
func (one) Error() text   { return "" }
func (one) Is(error) bool { return false }
func (one) As(any) bool   { return false }
func (one) Read() int     { return 0 }

type many struct{}

func (many) Unwrap() []error        { return nil }
func (many) Error(int) string       { return "" }
func (many) Is(any) bool            { return false }
func (many) As(*error) bool         { return false }
func (many) Format(string) string   { return "" }
func (many) String() (string, bool) { return "", false }
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
	ids := func(name string) map[string]string {
		out := map[string]string{}
		ms := types.NewMethodSet(pkg.Scope().Lookup(name).Type())
		for i := 0; i < ms.Len(); i++ {
			fn := ms.At(i).Obj().(*types.Func)
			out[fn.Name()] = MethodID(fn)
		}
		return out
	}
	for name, id := range ids("one") {
		if id != name {
			t.Errorf("one.%s = %q, want the plain name", name, id)
		}
	}
	for name, id := range ids("many") {
		if !strings.HasPrefix(id, name+"#") {
			t.Errorf("many.%s = %q, want a signature fingerprint", name, id)
		}
	}
}

package jsontype

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"
)

const src = `package p

type Inner struct {
	ID   int
	Name string
}

type Other struct {
	Name string ` + "`json:\"name\"`" + `
	Note string
}

type hidden struct {
	Deep string
}

type Outer struct {
	Inner
	*Other
	hidden
	Note  string
	Skip  int ` + "`json:\"-\"`" + `
	Dash  int ` + "`json:\"-,\"`" + `
	Quote int ` + "`json:\",string\"`" + `
	Ptr   *int ` + "`json:\"ptr,omitempty,string\"`" + `
	Slice []int ` + "`json:\",string\"`" + `
	low   int
}

type Values struct {
	A int
	B []byte
	C map[string][]*Inner
	D [2]bool
	E interface{}
	F func(int, ...string) (bool, error)
	G chan<- int
	H struct {
		X int ` + "`json:\"x\"`" + `
	}
}
`

func load(t *testing.T) *types.Package {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("p", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

type Inner struct {
	ID   int
	Name string
}

type Values struct {
	A int
	B []byte
	C map[string][]*Inner
	D [2]bool
	E interface{}
	F func(int, ...string) (bool, error)
	G chan<- int
	H struct {
		X int `json:"x"`
	}
}

func TestTypeStringMatchesReflect(t *testing.T) {
	pkg := load(t)
	st := pkg.Scope().Lookup("Values").Type().Underlying().(*types.Struct)
	rt := reflect.TypeOf(Values{})
	for i := 0; i < st.NumFields(); i++ {
		got := TypeString(st.Field(i).Type())
		want := strings.ReplaceAll(rt.Field(i).Type.String(), "jsontype.", "p.")
		if got != want {
			t.Errorf("field %s: got %q, want %q", st.Field(i).Name(), got, want)
		}
	}
}

func TestFields(t *testing.T) {
	pkg := load(t)
	var got []string
	for _, f := range Fields(pkg.Scope().Lookup("Outer").Type()) {
		s := f.Key
		if f.OmitEmpty {
			s += " omitempty"
		}
		if f.Quoted {
			s += " quoted"
		}
		if f.ViaPtr {
			s += " viaptr"
		}
		got = append(got, s)
	}
	want := []string{`"ID":`, `"Name":`, `"name": viaptr`, `"Deep":`, `"Note":`, `"-":`, `"Quote": quoted`, `"ptr": omitempty quoted`, `"Slice":`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("fields:\n got %v\nwant %v", got, want)
	}
}

func TestCheck(t *testing.T) {
	pkg := load(t)
	if p := Check(pkg.Scope().Lookup("Outer").Type()); p != nil {
		t.Errorf("Outer: unexpected problem %+v", p)
	}
	p := Check(pkg.Scope().Lookup("Values").Type())
	if p == nil || p.Path != "p.Values.F" {
		t.Errorf("Values: got %+v, want the func field rejected", p)
	}
}

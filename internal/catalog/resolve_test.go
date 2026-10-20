package catalog

import (
	"go/importer"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDeclarationPackagesShareImports(t *testing.T) {
	fsys := fstest.MapFS{
		"specs/declarations/left/left.go": {Data: []byte(`package left
import ("context"; "time")
func Read(ctx context.Context) (time.Time, error) { return time.Time{}, nil }
`)},
		"lib/right/right.go": {Data: []byte(`package right
import ("context"; "time")
func Echo(ctx context.Context) (time.Time, error) { return time.Time{}, nil }
`)},
	}
	fset := token.NewFileSet()
	loader := declarationLoader{fsys: fsys, fset: fset,
		imports: importer.ForCompiler(fset, "source", nil), packages: map[string]*types.Package{}}
	left, err := loader.lookupPackage(DeclarationModule + "left")
	if err != nil {
		t.Fatal(err)
	}
	right, err := loader.lookupPackage(LibModule + "right")
	if err != nil {
		t.Fatal(err)
	}
	if !types.Identical(left.Scope().Lookup("Read").Type(), right.Scope().Lookup("Echo").Type()) {
		t.Fatal("declarations imported separate identities for the same Go types")
	}
	timePackage, err := loader.lookupPackage("time")
	if err != nil {
		t.Fatal(err)
	}
	result := left.Scope().Lookup("Read").Type().(*types.Signature).Results().At(0).Type()
	if !types.Identical(result, timePackage.Scope().Lookup("Time").Type()) {
		t.Fatal("direct standard lookup did not reuse the dependency import")
	}
	if _, err := loader.lookupPackage("not/allowed"); err == nil {
		t.Fatal("accepted an unsupported declaration package")
	}
}

func TestResolversReloadDeclarationsAndCheckBodies(t *testing.T) {
	const name = "specs/declarations/probe/probe.go"
	fsys := fstest.MapFS{name: {Data: []byte("package probe\nfunc Value() int { return 1 }")}}
	resolve := func(want string) {
		t.Helper()
		in, out, err := Resolver(fsys)(DeclarationModule + "probe.Value")
		if err != nil || len(in) != 0 || !reflect.DeepEqual(out, []string{want}) {
			t.Fatalf("signature: %v, %v, %v", in, out, err)
		}
	}
	resolve("int")
	fsys[name].Data = []byte("package probe\nfunc Value() string { return \"changed\" }")
	resolve("string")
	fsys[name].Data = []byte("package probe\nfunc Value() string { return 1 }")
	if _, _, err := Resolver(fsys)(DeclarationModule + "probe.Value"); err == nil || !strings.Contains(err.Error(), "cannot use") {
		t.Fatalf("invalid declaration body was not checked: %v", err)
	}
}

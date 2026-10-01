package catalog

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path"
	"strings"

	"goalchemy/internal/contracts"
)

// stdPackages lists standard packages whose symbols may be mapped.
var stdPackages = map[string]bool{"errors": true}

// DeclarationModule is the module path under which specs/declarations lives.
const DeclarationModule = "goalchemy/specs/declarations/"

// Resolver type-checks Go declaration packages from the catalog filesystem
// and returns their signatures in contract type syntax.
func Resolver(fsys fs.FS) contracts.SymbolResolver {
	cache := map[string]*types.Package{}
	return func(symbol string) ([]string, []string, error) {
		i := strings.LastIndex(symbol, ".")
		if i < 0 {
			return nil, nil, fmt.Errorf("symbol %s is not package-qualified", symbol)
		}
		pkgPath, name := symbol[:i], symbol[i+1:]
		pkg, ok := cache[pkgPath]
		if !ok {
			var err error
			if strings.HasPrefix(symbol, DeclarationModule) {
				pkg, err = checkDir(fsys, "specs/declarations/"+strings.TrimPrefix(pkgPath, DeclarationModule), pkgPath)
			} else if stdPackages[pkgPath] {
				pkg, err = importer.Default().Import(pkgPath)
			} else {
				err = fmt.Errorf("symbol must be in a standard package or under %s", DeclarationModule)
			}
			if err != nil {
				return nil, nil, err
			}
			cache[pkgPath] = pkg
		}
		fn, ok := pkg.Scope().Lookup(name).(*types.Func)
		if !ok {
			return nil, nil, fmt.Errorf("%s is not a declared function", symbol)
		}
		sig := fn.Signature()
		return tupleTypes(sig.Params()), tupleTypes(sig.Results()), nil
	}
}

func tupleTypes(t *types.Tuple) []string {
	out := make([]string, t.Len())
	for i := range out {
		out[i] = types.TypeString(t.At(i).Type(), func(*types.Package) string { return "" })
	}
	return out
}

func checkDir(fsys fs.FS, dir, pkgPath string) (*types.Package, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		f, err := parser.ParseFile(fset, path.Join(dir, e.Name()), src, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	conf := types.Config{Importer: importer.Default(), GoVersion: "go1.25"}
	return conf.Check(pkgPath, fset, files, nil)
}

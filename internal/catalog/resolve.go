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
var stdPackages = map[string]bool{"errors": true, "sync": true, "runtime": true, "context": true, "time": true}

// LibModule is the import path prefix of Goalchemy's own capability
// packages, whose Go source doubles as their declarations.
const LibModule = "goalchemy/lib/"

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
		var pkg *types.Package
		if !strings.Contains(symbol, ".(") {
			var err error
			if pkg, err = lookupPackage(fsys, cache, pkgPath); err != nil {
				return nil, nil, err
			}
		}
		if j := strings.Index(symbol, ".("); j > 0 {
			// Method: pkg.(Type).Method; the receiver is the first input.
			pkgPath = symbol[:j]
			rest := symbol[j+2:]
			k := strings.Index(rest, ").")
			if k < 0 {
				return nil, nil, fmt.Errorf("malformed method symbol %s", symbol)
			}
			typeName, method := rest[:k], rest[k+2:]
			pkg, err := lookupPackage(fsys, cache, pkgPath)
			if err != nil {
				return nil, nil, err
			}
			tn, ok := pkg.Scope().Lookup(typeName).(*types.TypeName)
			if !ok {
				return nil, nil, fmt.Errorf("%s is not a type in %s", typeName, pkgPath)
			}
			var lookIn types.Type = types.NewPointer(tn.Type())
			if types.IsInterface(tn.Type()) {
				lookIn = tn.Type()
			}
			obj, _, _ := types.LookupFieldOrMethod(lookIn, true, pkg, method)
			fn, ok := obj.(*types.Func)
			if !ok {
				return nil, nil, fmt.Errorf("%s has no method %s", typeName, method)
			}
			sig := fn.Signature()
			recv := []string{TypeString(sig.Recv().Type())}
			return append(recv, tupleTypes(sig.Params())...), tupleTypes(sig.Results()), nil
		}
		switch o := pkg.Scope().Lookup(name).(type) {
		case *types.Func:
			sig := o.Signature()
			return tupleTypes(sig.Params()), tupleTypes(sig.Results()), nil
		case *types.Var:
			return nil, []string{TypeString(o.Type())}, nil
		}
		return nil, nil, fmt.Errorf("%s is not a declared function or variable", symbol)
	}
}

func lookupPackage(fsys fs.FS, cache map[string]*types.Package, pkgPath string) (*types.Package, error) {
	if pkg, ok := cache[pkgPath]; ok {
		return pkg, nil
	}
	var pkg *types.Package
	var err error
	if strings.HasPrefix(pkgPath, DeclarationModule) {
		pkg, err = checkDir(fsys, "specs/declarations/"+strings.TrimPrefix(pkgPath, DeclarationModule), pkgPath)
	} else if strings.HasPrefix(pkgPath, LibModule) {
		pkg, err = checkDir(fsys, "lib/"+strings.TrimPrefix(pkgPath, LibModule), pkgPath)
	} else if stdPackages[pkgPath] {
		pkg, err = importer.Default().Import(pkgPath)
	} else {
		err = fmt.Errorf("package %s is neither standard nor under %s or %s", pkgPath, DeclarationModule, LibModule)
	}
	if err != nil {
		return nil, err
	}
	cache[pkgPath] = pkg
	return pkg, nil
}

// TypeString renders a type in contract syntax: package-qualified by name.
func TypeString(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}

func tupleTypes(t *types.Tuple) []string {
	out := make([]string, t.Len())
	for i := range out {
		out[i] = TypeString(t.At(i).Type())
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
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil), GoVersion: "go1.25"}
	return conf.Check(pkgPath, fset, files, nil)
}

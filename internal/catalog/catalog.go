// Package catalog locates the contract catalog and exposes the external
// capability registry derived from it.
package catalog

import (
	"go/types"
	"io/fs"
	"os"
	"strings"

	"goalchemy"
	"goalchemy/internal/contracts"
	"goalchemy/internal/diagnostics"
)

// FS returns the catalog root: GOALCHEMY_ROOT when set, else the embedded copy.
func FS() fs.FS {
	if root := os.Getenv("GOALCHEMY_ROOT"); root != "" {
		return os.DirFS(root)
	}
	return goalchemy.Assets
}

func Load(fsys fs.FS) (*contracts.Catalog, []diagnostics.Diagnostic) {
	return contracts.Load(fsys, contracts.LoadOptions{Resolve: Resolver(fsys)})
}

// Registry maps external Go symbols to capability contract IDs.
type Registry struct {
	Symbols  map[string]string
	Packages map[string]bool
}

func NewRegistry(cat *contracts.Catalog) *Registry {
	r := &Registry{Symbols: map[string]string{}, Packages: map[string]bool{}}
	if cat == nil {
		return r
	}
	for id, f := range cat.Functions {
		if f.Signature.Source != "declaration" {
			continue
		}
		sym := f.Signature.Symbol
		r.Symbols[sym] = id
		if i := strings.LastIndex(sym, "."); i > 0 {
			pkg := sym[:i]
			if j := strings.Index(pkg, ".("); j > 0 {
				pkg = pkg[:j]
			}
			r.Packages[pkg] = true
		}
	}
	return r
}

// SymbolKey renders the registry key for an object: pkg.Func or pkg.(Type).Method.
func SymbolKey(obj types.Object) string {
	if obj.Pkg() == nil {
		return obj.Name()
	}
	if fn, ok := obj.(*types.Func); ok {
		if recv := fn.Signature().Recv(); recv != nil {
			t := recv.Type()
			if p, ok := t.(*types.Pointer); ok {
				t = p.Elem()
			}
			if n, ok := t.(*types.Named); ok {
				return obj.Pkg().Path() + ".(" + n.Obj().Name() + ")." + fn.Name()
			}
		}
	}
	return obj.Pkg().Path() + "." + obj.Name()
}

func (r *Registry) External(obj types.Object) bool {
	_, ok := r.Symbols[SymbolKey(obj)]
	return ok
}

func (r *Registry) Package(path string) bool { return r.Packages[path] }

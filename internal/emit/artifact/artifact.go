// Package artifact describes native source files and source-package ownership
// for one whole-program emission. It does not assign new source identities.
package artifact

import (
	"crypto/sha256"
	"fmt"
	"go/token"
	"path"
	"sort"

	"github.com/eugenioenko/goalchemy/internal/ir"
	"github.com/eugenioenko/goalchemy/internal/naming"
	"github.com/eugenioenko/goalchemy/internal/sourcemap"
)

// File is one native source artifact. Owner is an included Go import path;
// the empty owner denotes central entry, API or canonical shared support.
// Lines uses physical 1-based generated line numbers. SourceMap uses v3 maps.
type File struct {
	Path      string
	Source    []byte
	Owner     string
	Lines     map[int]token.Position
	SourceMap *sourcemap.Map
}

// PackageFiles records the source graph and its emitted native source members.
// Shared coordinator/support files have no source package owner.
type PackageFiles struct {
	ir.Package
	Files []string `json:"files"`
}

func (l *Layout) PackageFiles(files []File) []PackageFiles {
	packages := make([]PackageFiles, 0, len(l.Packages))
	for _, pkg := range l.Packages {
		item := PackageFiles{Package: pkg, Files: []string{}}
		for _, file := range files {
			if file.Owner == pkg.Path {
				item.Files = append(item.Files, file.Path)
			}
		}
		packages = append(packages, item)
	}
	return packages
}

// Layout shares a single ownership index across every declaration in emission.
// Packages follows source dependency order; stems never depend on naming mode.
type Layout struct {
	Packages []ir.Package
	stems    map[string]string
	reps     map[*ir.Type]string
}

func NewLayout(p *ir.Program) (*Layout, error) {
	l := &Layout{Packages: append([]ir.Package(nil), p.Packages...), stems: map[string]string{}, reps: map[*ir.Type]string{}}
	// Small hand-built IR programs may omit frontend metadata. Keep those
	// emitters usable while deriving owners only from explicit IR package paths.
	seen := map[string]bool{}
	for _, pkg := range l.Packages {
		if pkg.Path == "" || seen[pkg.Path] {
			return nil, fmt.Errorf("invalid or duplicate source package %q", pkg.Path)
		}
		seen[pkg.Path] = true
	}
	missing := map[string]bool{}
	add := func(owner string) {
		if owner != "" && !seen[owner] {
			missing[owner] = true
		}
	}
	for _, f := range p.Funcs {
		add(f.Pkg)
	}
	for _, g := range p.Globals {
		add(g.Pkg)
	}
	if p.Types != nil && len(p.Packages) == 0 {
		for _, t := range p.Types.All {
			if t.Kind == ir.KNamed {
				add(t.Pkg)
			}
		}
	}
	var paths []string
	for owner := range missing {
		paths = append(paths, owner)
	}
	sort.Strings(paths)
	if len(p.Packages) > 0 && len(paths) > 0 {
		return nil, fmt.Errorf("declaration owner %q is not an included source package", paths[0])
	}
	for _, owner := range paths {
		l.Packages = append(l.Packages, ir.Package{Path: owner, Name: path.Base(owner), Imports: []string{}})
	}
	used := map[string]string{}
	for _, pkg := range l.Packages {
		base := naming.Identifier(path.Base(pkg.Path))
		if len(base) > 24 {
			base = base[:24]
		}
		hash := sha256.Sum256([]byte(pkg.Path))
		stem := fmt.Sprintf("pkg_%s_%x", base, hash[:8])
		if previous, ok := used[stem]; ok {
			return nil, fmt.Errorf("source package filename collision: %s and %s", previous, pkg.Path)
		}
		used[stem], l.stems[pkg.Path] = pkg.Path, stem
	}
	// Match the naming index's canonical named representative for underlying
	// types. This records ownership without duplicating shared representations.
	if p.Types != nil {
		representatives := map[*ir.Type]*ir.Type{}
		for _, t := range p.Types.All {
			if t.Kind != ir.KNamed || t.Pkg == "" {
				continue
			}
			old := representatives[t.U()]
			if old == nil || t.Name+"\x00"+t.Pkg < old.Name+"\x00"+old.Pkg {
				representatives[t.U()] = t
			}
		}
		for underlying, named := range representatives {
			l.reps[underlying] = l.TypeOwner(named)
		}
	}
	return l, nil
}

func (l *Layout) Stem(owner string) string { return l.stems[owner] }

// TypeOwner is the declared source owner; anonymous types are central.
func (l *Layout) TypeOwner(t *ir.Type) string {
	if t.Kind == ir.KNamed && l.stems[t.Pkg] != "" {
		return t.Pkg
	}
	return ""
}

// RepresentationOwner selects one owner for helpers keyed by underlying type.
func (l *Layout) RepresentationOwner(t *ir.Type) string { return l.reps[t.U()] }

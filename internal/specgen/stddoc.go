package specgen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"path"
	"sort"
	"strings"
)

type stdPackage struct {
	path     string
	synopsis string
	doc      *doc.Package
	fset     *token.FileSet
}

func loadStdPackages(fsys fs.FS) ([]*stdPackage, error) {
	var out []*stdPackage
	err := fs.WalkDir(fsys, "std", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == "std" {
				return fs.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		sp, err := loadStdPackage(fsys, p)
		if err != nil {
			return err
		}
		if sp != nil {
			out = append(out, sp)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, err
}

func loadStdPackage(fsys fs.FS, dir string) (*stdPackage, error) {
	ents, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, ent := range ents {
		name := ent.Name()
		if ent.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if bytes.HasPrefix(src, []byte("//go:build ignore")) {
			continue
		}
		f, err := parser.ParseFile(fset, path.Join(dir, name), src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, nil
	}
	rel := strings.TrimPrefix(dir, "std/")
	dp, err := doc.NewFromFiles(fset, files, stdModule+rel)
	if err != nil {
		return nil, err
	}
	return &stdPackage{path: rel, synopsis: dp.Synopsis(dp.Doc), doc: dp, fset: fset}, nil
}

func anchor(stdPath string) string { return "std" + strings.ReplaceAll(stdPath, "/", "") }

func (sp *stdPackage) node(n any) string {
	var b bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	if err := cfg.Fprint(&b, sp.fset, n); err != nil {
		return ""
	}
	return b.String()
}

func (sp *stdPackage) funcDecl(f *doc.Func) string {
	d := *f.Decl
	d.Doc, d.Body = nil, nil
	return sp.node(&d)
}

func (sp *stdPackage) genDecl(g *ast.GenDecl) string {
	d := *g
	d.Doc = nil
	if d.Tok == token.TYPE {
		specs := make([]ast.Spec, len(d.Specs))
		for i, s := range d.Specs {
			ts := *s.(*ast.TypeSpec)
			ts.Doc, ts.Comment = nil, nil
			if st, ok := ts.Type.(*ast.StructType); ok && !hasExportedField(st) {
				ts.Type = &ast.StructType{Fields: &ast.FieldList{}}
			}
			specs[i] = &ts
		}
		d.Specs = specs
	}
	return sp.node(&d)
}

func hasExportedField(st *ast.StructType) bool {
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			if n.IsExported() {
				return true
			}
		}
	}
	return false
}

func docText(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func writeStdPackage(b *bytes.Buffer, sp *stdPackage) {
	fmt.Fprintf(b, "\n## std/%s\n\n```go\nimport \"%s%s\"\n```\n\n", sp.path, stdModule, sp.path)
	if t := docText(sp.doc.Doc); t != "" {
		b.WriteString(t + "\n\n")
	}
	for _, v := range append(append([]*doc.Value{}, sp.doc.Consts...), sp.doc.Vars...) {
		fmt.Fprintf(b, "```go\n%s\n```\n\n", sp.genDecl(v.Decl))
		if t := docText(v.Doc); t != "" {
			b.WriteString(t + "\n\n")
		}
	}
	for _, f := range sp.doc.Funcs {
		writeStdFunc(b, sp, f.Name, f)
	}
	for _, t := range sp.doc.Types {
		if !ast.IsExported(t.Name) {
			continue
		}
		fmt.Fprintf(b, "### type %s\n\n```go\n%s\n```\n\n", t.Name, sp.genDecl(t.Decl))
		if d := docText(t.Doc); d != "" {
			b.WriteString(d + "\n\n")
		}
		for _, v := range append(append([]*doc.Value{}, t.Consts...), t.Vars...) {
			fmt.Fprintf(b, "```go\n%s\n```\n\n", sp.genDecl(v.Decl))
			if d := docText(v.Doc); d != "" {
				b.WriteString(d + "\n\n")
			}
		}
		for _, f := range t.Funcs {
			writeStdFunc(b, sp, f.Name, f)
		}
		for _, m := range t.Methods {
			writeStdFunc(b, sp, t.Name+"."+m.Name, m)
		}
	}
}

func writeStdFunc(b *bytes.Buffer, sp *stdPackage, name string, f *doc.Func) {
	fmt.Fprintf(b, "### %s\n\n```go\n%s\n```\n\n", name, sp.funcDecl(f))
	if d := docText(f.Doc); d != "" {
		b.WriteString(d + "\n\n")
	}
}

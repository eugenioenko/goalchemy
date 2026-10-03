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
	"unicode"
	"unicode/utf8"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

// LibraryDoc is the generated runtime library reference.
const LibraryDoc = "docs/library.md"

const libModule = "github.com/eugenioenko/goalchemy/lib/"

type libEntry struct {
	name   string
	member string
	fc     *contracts.Function
}

type libPackage struct {
	name     string
	synopsis string
	entries  []libEntry
	funcs    map[string]*ast.FuncDecl
	vars     map[string]bool
	fset     *token.FileSet
}

func libraryDoc(cat *contracts.Catalog) ([]byte, error) {
	pkgs := map[string]*libPackage{}
	for _, fc := range cat.Functions {
		if fc.Signature.Source != "declaration" {
			continue
		}
		sym := strings.TrimPrefix(fc.Signature.Symbol, libModule)
		i := strings.Index(sym, ".")
		if i <= 0 {
			continue
		}
		name, member := sym[:i], sym[i+1:]
		p, ok := pkgs[name]
		if !ok {
			var err error
			if p, err = loadLibPackage(cat.FS, name); err != nil {
				return nil, err
			}
			if p == nil {
				continue
			}
			pkgs[name] = p
		}
		p.entries = append(p.entries, libEntry{name: strings.NewReplacer("(", "", ")", "").Replace(member), member: member, fc: fc})
	}
	names := make([]string, 0, len(pkgs))
	for n := range pkgs {
		names = append(names, n)
	}
	sort.Strings(names)
	targets := make([]string, 0, len(cat.Targets))
	for n := range cat.Targets {
		targets = append(targets, n)
	}
	sort.Strings(targets)

	var b bytes.Buffer
	fmt.Fprintf(&b, "<!-- %s -->\n\n", Marker)
	b.WriteString("# Runtime library reference\n\n")
	b.WriteString("Goalchemy programs get their capabilities from the packages under `" + strings.TrimSuffix(libModule, "/") + "`. ")
	b.WriteString("Each package is ordinary Go, so a program also builds and runs with the Go toolchain; the compiler maps every call to the target runtime implementation of its contract.\n\n")
	b.WriteString("This file is generated from the contracts in `specs/runtime` and the target mappings in `targets/*/target.yaml`. Edit those and run `make spec-generate`.\n\n")
	b.WriteString("Every entry lists:\n\n")
	b.WriteString("- **Gate**: `sequential` means the function never suspends, so it suits sequential programs and synchronous exports. `cooperative` means the contract lets the call suspend the calling task (a scheduler pause point); write such code for the cooperative execution model, where other tasks run while it waits.\n")
	b.WriteString("- **Bounds** and **Errors**: the limits checked before work starts and what is returned when they or the operation fail.\n")
	b.WriteString("- **Contract**: the contract ID and version under `specs/runtime`.\n\n")
	b.WriteString("## Native dependencies\n\n")
	b.WriteString("| Target | Status | Minimum | Native dependencies |\n| --- | --- | --- | --- |\n")
	for _, n := range targets {
		t := cat.Targets[n]
		native := t.Environment.Native
		if native == "" {
			native = "None declared."
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", n, t.Status, cell(t.Environment.Minimum), cell(native))
	}
	b.WriteString("\n## Packages\n\n| Package | Import path | Summary |\n| --- | --- | --- |\n")
	for _, n := range names {
		fmt.Fprintf(&b, "| [%s](#%s) | `%s%s` | %s |\n", n, n, libModule, n, cell(pkgs[n].synopsis))
	}
	for _, n := range names {
		p := pkgs[n]
		sort.Slice(p.entries, func(i, j int) bool { return p.entries[i].name < p.entries[j].name })
		fmt.Fprintf(&b, "\n## %s\n\n```go\nimport \"%s%s\"\n```\n\n", n, libModule, n)
		if p.synopsis != "" {
			b.WriteString(p.synopsis + "\n\n")
		}
		b.WriteString("| Function |")
		for _, t := range targets {
			b.WriteString(" " + t + " |")
		}
		b.WriteString("\n| --- |" + strings.Repeat(" :-: |", len(targets)) + "\n")
		for _, e := range p.entries {
			fmt.Fprintf(&b, "| `%s` |", e.name)
			for _, t := range targets {
				mark := "no"
				if _, ok := cat.Targets[t].Function(e.fc.ID); ok {
					mark = "yes"
				}
				b.WriteString(" " + mark + " |")
			}
			b.WriteString("\n")
		}
		for _, e := range p.entries {
			fc := e.fc
			fmt.Fprintf(&b, "\n### %s\n\n```go\n%s\n```\n\n%s\n\n", e.name, p.signature(n, e), summary(fc.Summary))
			gate := "`sequential` (never suspends)"
			if fc.Suspension == "may" {
				gate = "`cooperative` (may suspend the calling task)"
			}
			fmt.Fprintf(&b, "- **Gate**: %s\n- **Determinism**: %s\n", gate, fc.Determinism)
			for _, k := range []struct{ label, key string }{{"Bounds", "bounds"}, {"Errors", "errors"}} {
				if v := fc.Behavior[k.key]; v != "" && v != "Not applicable." {
					fmt.Fprintf(&b, "- **%s**: %s\n", k.label, v)
				}
			}
			fmt.Fprintf(&b, "- **Contract**: `%s` %s\n", fc.ID, fc.Version)
		}
	}
	return b.Bytes(), nil
}

func loadLibPackage(fsys fs.FS, name string) (*libPackage, error) {
	dir := path.Join("lib", name)
	ents, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, nil
	}
	p := &libPackage{name: name, funcs: map[string]*ast.FuncDecl{}, vars: map[string]bool{}, fset: token.NewFileSet()}
	for _, ent := range ents {
		fn := ent.Name()
		if ent.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		src, err := fs.ReadFile(fsys, path.Join(dir, fn))
		if err != nil {
			return nil, err
		}
		f, err := parser.ParseFile(p.fset, path.Join(dir, fn), src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if f.Doc != nil && p.synopsis == "" {
			var lines []string
			for _, l := range strings.Split(f.Doc.Text(), "\n") {
				if !strings.HasPrefix(l, "SPDX-License-Identifier:") {
					lines = append(lines, l)
				}
			}
			p.synopsis = new(doc.Package).Synopsis(strings.Join(lines, "\n"))
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				key := d.Name.Name
				if d.Recv != nil && len(d.Recv.List) == 1 {
					t := d.Recv.List[0].Type
					if s, ok := t.(*ast.StarExpr); ok {
						t = s.X
					}
					if id, ok := t.(*ast.Ident); ok {
						key = "(" + id.Name + ")." + key
					}
				}
				p.funcs[key] = d
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					for _, id := range s.(*ast.ValueSpec).Names {
						p.vars[id.Name] = true
					}
				}
			}
		}
	}
	return p, nil
}

func (p *libPackage) signature(pkg string, e libEntry) string {
	fc := e.fc
	if d, ok := p.funcs[e.member]; ok {
		decl := *d
		decl.Doc, decl.Body = nil, nil
		var b bytes.Buffer
		if err := printer.Fprint(&b, p.fset, &decl); err == nil {
			return b.String()
		}
	}
	local := func(t string) string { return strings.ReplaceAll(t, pkg+".", "") }
	outs := make([]string, 0, len(fc.Signature.Outputs))
	for _, o := range fc.Signature.Outputs {
		outs = append(outs, local(o.Type))
	}
	if p.vars[e.member] && len(fc.Signature.Inputs) == 0 && len(outs) == 1 {
		return "var " + e.member + " " + outs[0]
	}
	ins := fc.Signature.Inputs
	head := "func "
	name := e.member
	if i := strings.Index(name, ")."); strings.HasPrefix(name, "(") && i > 0 && len(ins) > 0 {
		head += "(" + camel(ins[0].Name) + " " + local(ins[0].Type) + ") "
		name, ins = name[i+2:], ins[1:]
	}
	params := make([]string, 0, len(ins))
	for i, in := range ins {
		t := local(in.Type)
		if fc.Signature.Variadic && i == len(ins)-1 {
			t = "..." + strings.TrimPrefix(t, "[]")
		}
		params = append(params, camel(in.Name)+" "+t)
	}
	res := ""
	switch len(outs) {
	case 0:
	case 1:
		res = " " + outs[0]
	default:
		res = " (" + strings.Join(outs, ", ") + ")"
	}
	return head + name + "(" + strings.Join(params, ", ") + ")" + res
}

func summary(s string) string {
	if i := strings.Index(s, ": "); i > 0 && !strings.ContainsAny(s[:i], " ") {
		s = s[i+2:]
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

func camel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

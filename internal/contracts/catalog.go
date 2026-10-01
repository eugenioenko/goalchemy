package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"goalchemy/internal/diagnostics"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// Catalog is the normalized contract catalog: canonical types, runtime
// function contracts, and target mappings, all validated.
type Catalog struct {
	FS        fs.FS
	Types     map[string]*Type
	Functions map[string]*Function
	Targets   map[string]*Target
}

// SymbolResolver checks declaration-backed signatures against Go
// declarations. It returns the Go-derived signature types or an error.
type SymbolResolver func(symbol string) (inputs, outputs []string, err error)

type LoadOptions struct {
	Resolve SymbolResolver
}

func diag(file, code, format string, args ...any) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{Code: code, Severity: diagnostics.Error, File: file,
		Feature: "contract catalog", Message: fmt.Sprintf(format, args...),
		Remedy: "Correct the contract or mapping and run goalchemy spec validate."}
}

// Load reads specs/ and targets/ from fsys, validates every document against
// its JSON Schema, then performs semantic validation across files.
func Load(fsys fs.FS, opts LoadOptions) (*Catalog, []diagnostics.Diagnostic) {
	cat := &Catalog{FS: fsys, Types: map[string]*Type{}, Functions: map[string]*Function{}, Targets: map[string]*Target{}}
	var ds []diagnostics.Diagnostic
	schemas := map[string]*jsonschema.Schema{}
	for _, kind := range []string{"type", "function", "target"} {
		p := "specs/schema/" + kind + ".schema.json"
		s, err := compileSchema(fsys, p)
		if err != nil {
			ds = append(ds, diag(p, "GCC001", "schema %s: %v", kind, err))
			continue
		}
		schemas[kind] = s
	}
	if len(ds) > 0 {
		return cat, ds
	}
	load := func(p, kind string, into any) (string, bool) {
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			ds = append(ds, diag(p, "GCC002", "%v", err))
			return "", false
		}
		v, err := ParseYAML(raw)
		if err != nil {
			ds = append(ds, diag(p, "GCC002", "invalid contract YAML: %v", err))
			return "", false
		}
		canon, _ := json.Marshal(v)
		jv, err := jsonschema.UnmarshalJSON(bytes.NewReader(canon))
		if err == nil {
			err = schemas[kind].Validate(jv)
		}
		if err != nil {
			ds = append(ds, diag(p, "GCC003", "schema violation: %s", oneLine(err)))
			return "", false
		}
		if err := json.Unmarshal(canon, into); err != nil {
			ds = append(ds, diag(p, "GCC003", "decode: %v", err))
			return "", false
		}
		sum := sha256.Sum256(canon)
		return hex.EncodeToString(sum[:]), true
	}
	for _, p := range yamlFiles(fsys, "specs/types") {
		t := &Type{}
		if h, ok := load(p, "type", t); ok {
			t.Path, t.Hash = p, h
			if prev, dup := cat.Types[t.ID]; dup {
				ds = append(ds, diag(p, "GCC004", "duplicate type contract %s (also %s)", t.ID, prev.Path))
				continue
			}
			cat.Types[t.ID] = t
		}
	}
	for _, p := range yamlFiles(fsys, "specs/runtime") {
		f := &Function{}
		if h, ok := load(p, "function", f); ok {
			f.Path, f.Hash = p, h
			if prev, dup := cat.Functions[f.ID]; dup {
				ds = append(ds, diag(p, "GCC004", "duplicate function contract %s (also %s)", f.ID, prev.Path))
				continue
			}
			cat.Functions[f.ID] = f
		}
	}
	targetDirs, _ := fs.ReadDir(fsys, "targets")
	for _, d := range targetDirs {
		p := "targets/" + d.Name() + "/target.yaml"
		if _, err := fs.Stat(fsys, p); err != nil {
			continue
		}
		t := &Target{}
		if _, ok := load(p, "target", t); ok {
			t.Path, t.Dir = p, "targets/"+d.Name()
			if t.Target != d.Name() {
				ds = append(ds, diag(p, "GCC005", "target %q is declared in directory %q", t.Target, d.Name()))
			}
			cat.Targets[t.Target] = t
		}
	}
	ds = append(ds, cat.validate(opts)...)
	diagnostics.Sort(ds)
	return cat, ds
}

func oneLine(err error) string {
	s := err.Error()
	s = strings.ReplaceAll(s, "\n", "; ")
	return strings.Join(strings.Fields(s), " ")
}

func compileSchema(fsys fs.FS, p string) (*jsonschema.Schema, error) {
	raw, err := fs.ReadFile(fsys, p)
	if err != nil {
		return nil, err
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	url := "file:///" + p
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	return c.Compile(url)
}

func yamlFiles(fsys fs.FS, root string) []string {
	var out []string
	_ = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".yaml") {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func (cat *Catalog) version(id string) (string, bool) {
	if t, ok := cat.Types[id]; ok {
		return t.Version, true
	}
	if f, ok := cat.Functions[id]; ok {
		return f.Version, true
	}
	return "", false
}

func (cat *Catalog) validate(opts LoadOptions) []diagnostics.Diagnostic {
	var ds []diagnostics.Diagnostic
	for _, id := range sortedKeys(cat.Types) {
		t := cat.Types[id]
		if t.ID != "type."+t.Family {
			ds = append(ds, diag(t.Path, "GCC005", "type contract %s must be named type.%s", t.ID, t.Family))
		}
		for _, f := range t.ExercisedBy {
			fn, ok := cat.Functions[f]
			if !ok {
				ds = append(ds, diag(t.Path, "GCC005", "exercised_by names unknown function %s", f))
				continue
			}
			if !fn.dependsOn(t.ID) {
				ds = append(ds, diag(t.Path, "GCC005", "%s is listed as exercising %s but does not depend on it", f, t.ID))
			}
		}
	}
	for _, id := range sortedKeys(cat.Functions) {
		ds = append(ds, cat.validateFunction(cat.Functions[id], opts)...)
	}
	ds = append(ds, cat.cycles()...)
	for _, name := range sortedKeys(cat.Targets) {
		ds = append(ds, cat.validateTarget(cat.Targets[name])...)
	}
	return ds
}

func (f *Function) dependsOn(id string) bool {
	for _, d := range f.DependsOn {
		if d.ID == id {
			return true
		}
	}
	return false
}

func (cat *Catalog) validateFunction(f *Function, opts LoadOptions) []diagnostics.Diagnostic {
	var ds []diagnostics.Diagnostic
	add := func(code, format string, args ...any) {
		ds = append(ds, diag(f.Path, code, "%s: "+format, append([]any{f.ID}, args...)...))
	}
	deps := map[string]bool{}
	for _, d := range f.DependsOn {
		if deps[d.ID] {
			add("GCC005", "duplicate dependency %s", d.ID)
		}
		deps[d.ID] = true
		v, ok := cat.version(d.ID)
		switch {
		case !ok:
			add("GCC006", "unknown dependency %s", d.ID)
		case v != d.Version:
			add("GCC006", "dependency %s requires version %s, catalog has %s", d.ID, d.Version, v)
		}
	}
	vars := map[string]TypeParam{}
	for _, tp := range f.Signature.TypeParameters {
		if _, dup := vars[tp.Name]; dup {
			add("GCC005", "duplicate type parameter %s", tp.Name)
		}
		vars[tp.Name] = tp
		if tp.Family != "any" {
			if _, ok := cat.Types[tp.Family]; !ok {
				add("GCC006", "type parameter %s uses unknown family %s", tp.Name, tp.Family)
			} else if !deps[tp.Family] {
				add("GCC006", "type parameter family %s must be listed in depends_on", tp.Family)
			}
		}
	}
	used := map[string]bool{}
	sigTypes := map[string]*TypeExpr{}
	checkParams := func(ps []Param, what string) {
		seen := map[string]bool{}
		for _, p := range ps {
			if seen[p.Name] {
				add("GCC005", "duplicate %s %s", what, p.Name)
			}
			seen[p.Name] = true
			te, err := ParseType(p.Type)
			if err != nil {
				add("GCC006", "%s %s: %v", what, p.Name, err)
				continue
			}
			if err := checkTypeNames(te, vars, used); err != nil {
				add("GCC006", "%s %s: %v", what, p.Name, err)
				continue
			}
			sigTypes[what+":"+p.Name] = te
		}
	}
	checkParams(f.Signature.Inputs, "input")
	checkParams(f.Signature.Outputs, "output")
	for name := range vars {
		if !used[name] {
			add("GCC006", "type parameter %s is not used by the signature", name)
		}
	}
	if f.Signature.Source == "declaration" {
		switch {
		case len(vars) > 0:
			add("GCC006", "declaration-backed contracts cannot have type parameters")
		case opts.Resolve == nil:
			add("GCC007", "declaration symbol %s cannot be resolved without the Go loader", f.Signature.Symbol)
		default:
			in, out, err := opts.Resolve(f.Signature.Symbol)
			if err != nil {
				add("GCC007", "declaration symbol %s: %v", f.Signature.Symbol, err)
			} else {
				if !sameTypes(in, f.Signature.Inputs) {
					add("GCC007", "declared inputs %v do not match contract inputs", in)
				}
				if !sameTypes(out, f.Signature.Outputs) {
					add("GCC007", "declared outputs %v do not match contract outputs", out)
				}
			}
		}
	}
	names := map[string]bool{}
	for _, c := range f.Cases {
		if names[c.Name] {
			add("GCC005", "duplicate case %s", c.Name)
		}
		names[c.Name] = true
		for _, e := range validateCase(f, c, vars) {
			add("GCC008", "case %s: %s", c.Name, e)
		}
	}
	return ds
}

func sameTypes(decl []string, ps []Param) bool {
	if len(decl) != len(ps) {
		return false
	}
	for i := range decl {
		if decl[i] != ps[i].Type {
			return false
		}
	}
	return true
}

func checkTypeNames(t *TypeExpr, vars map[string]TypeParam, used map[string]bool) error {
	switch t.Kind {
	case "name":
		if _, ok := vars[t.Name]; ok {
			used[t.Name] = true
			return nil
		}
		if !t.Concrete() {
			return fmt.Errorf("unknown type %s", t.Name)
		}
		return nil
	case "map":
		if err := checkTypeNames(t.Key, vars, used); err != nil {
			return err
		}
	}
	return checkTypeNames(t.Elem, vars, used)
}

func validateCase(f *Function, c Case, vars map[string]TypeParam) []string {
	var errs []string
	fail := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	args := map[string]*TypeExpr{}
	for name, tp := range vars {
		s, ok := c.Types[name]
		if !ok {
			fail("missing type argument %s", name)
			continue
		}
		te, err := ParseType(s)
		if err != nil {
			fail("type argument %s: %v", name, err)
			continue
		}
		if !InFamily(te, tp.Family) {
			fail("type argument %s=%s is not in family %s", name, s, tp.Family)
		}
		if tp.Comparable && !te.Comparable() {
			fail("type argument %s=%s is not comparable", name, s)
		}
		args[name] = te
	}
	for name := range c.Types {
		if _, ok := vars[name]; !ok {
			fail("unknown type argument %s", name)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	scope := Scope{}
	for _, b := range c.Let {
		if _, dup := scope[b.Name]; dup {
			fail("duplicate binding %s", b.Name)
			continue
		}
		te, err := ParseType(b.Type)
		if err != nil {
			fail("binding %s: %v", b.Name, err)
			continue
		}
		te = te.Substitute(args)
		if !te.Concrete() {
			fail("binding %s has non-concrete type %s", b.Name, te)
			continue
		}
		if _, err := DecodeValue(b.Value, te, scope); err != nil {
			fail("binding %s: %v", b.Name, err)
		}
		scope[b.Name] = te
	}
	ins := f.Signature.Inputs
	if len(c.Call) != len(ins) {
		fail("call passes %d arguments, signature has %d", len(c.Call), len(ins))
	} else {
		for i, a := range c.Call {
			bt, ok := scope[a]
			if !ok {
				fail("call argument %s is not bound", a)
				continue
			}
			want, _ := ParseType(ins[i].Type)
			if want == nil {
				continue
			}
			if w := want.Substitute(args); w.String() != bt.String() {
				fail("argument %s has type %s, input %s needs %s", a, bt, ins[i].Name, w)
			}
		}
	}
	if c.Expect.Results != nil {
		outs := f.Signature.Outputs
		if len(c.Expect.Results) != len(outs) {
			fail("expects %d results, signature has %d", len(c.Expect.Results), len(outs))
		} else {
			for i, r := range c.Expect.Results {
				ot, _ := ParseType(outs[i].Type)
				if ot == nil {
					continue
				}
				if _, err := DecodeValue(r, ot.Substitute(args), scope); err != nil {
					fail("result %s: %v", outs[i].Name, err)
				}
			}
		}
	}
	if c.Expect.Panic != nil {
		if !contains(f.PanicCategories, c.Expect.Panic.Category) {
			fail("panic category %q is not declared by the contract", c.Expect.Panic.Category)
		}
	}
	for _, a := range c.Expect.After {
		bt, ok := scope[a.Name]
		if !ok {
			fail("postcondition names unknown binding %s", a.Name)
			continue
		}
		if _, err := DecodeValue(a.Value, bt, scope); err != nil {
			fail("postcondition %s: %v", a.Name, err)
		}
	}
	return errs
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func (cat *Catalog) cycles() []diagnostics.Diagnostic {
	var ds []diagnostics.Diagnostic
	state := map[string]int{}
	var visit func(id string, stack []string)
	visit = func(id string, stack []string) {
		state[id] = 1
		f := cat.Functions[id]
		for _, d := range f.DependsOn {
			if _, ok := cat.Functions[d.ID]; !ok {
				continue
			}
			switch state[d.ID] {
			case 1:
				ds = append(ds, diag(f.Path, "GCC006", "dependency cycle: %s -> %s", strings.Join(append(stack, id), " -> "), d.ID))
			case 0:
				visit(d.ID, append(stack, id))
			}
		}
		state[id] = 2
	}
	for _, id := range sortedKeys(cat.Functions) {
		if state[id] == 0 {
			visit(id, nil)
		}
	}
	return ds
}

func (cat *Catalog) validateTarget(t *Target) []diagnostics.Diagnostic {
	var ds []diagnostics.Diagnostic
	add := func(code, format string, args ...any) {
		ds = append(ds, diag(t.Path, code, "target %s: "+format, append([]any{t.Target}, args...)...))
	}
	seen := map[string]bool{}
	for _, r := range t.Representations {
		if seen[r.Type] {
			add("GCC005", "duplicate representation for %s", r.Type)
		}
		seen[r.Type] = true
		tc, ok := cat.Types[r.Type]
		switch {
		case !ok:
			add("GCC006", "representation for unknown type contract %s", r.Type)
		case tc.Version != r.ContractVersion:
			add("GCC006", "representation for %s targets version %s, catalog has %s", r.Type, r.ContractVersion, tc.Version)
		}
	}
	seen = map[string]bool{}
	files := map[string]string{}
	for _, f := range t.Functions {
		if seen[f.ID] {
			add("GCC005", "duplicate implementation for %s", f.ID)
		}
		seen[f.ID] = true
		fc, ok := cat.Functions[f.ID]
		switch {
		case !ok:
			add("GCC006", "implementation for unknown function contract %s", f.ID)
		case fc.Version != f.ContractVersion:
			add("GCC006", "implementation of %s targets version %s, catalog has %s", f.ID, f.ContractVersion, fc.Version)
		}
		if prev, dup := files[f.Implementation]; dup {
			add("GCC005", "%s and %s share implementation file %s", prev, f.ID, f.Implementation)
		}
		files[f.Implementation] = f.ID
		if !fs.ValidPath(f.Implementation) || strings.HasPrefix(f.Implementation, "..") {
			add("GCC006", "implementation path %s must be relative to the target directory", f.Implementation)
		} else if _, err := fs.Stat(cat.FS, path.Join(t.Dir, f.Implementation)); err != nil {
			add("GCC006", "implementation file %s for %s is missing", f.Implementation, f.ID)
		}
		for _, r := range f.Requires {
			if _, ok := t.Function(r); !ok {
				add("GCC006", "%s requires %s, which the target does not implement", f.ID, r)
			}
		}
		if ok {
			for _, d := range fc.DependsOn {
				if _, isFn := cat.Functions[d.ID]; isFn {
					if _, has := t.Function(d.ID); !has {
						add("GCC006", "%s depends on %s, which the target does not implement", f.ID, d.ID)
					}
				}
			}
		}
	}
	if t.Status != "planned" {
		for _, tc := range sortedKeys(cat.Types) {
			if !seen2(t.Representations, tc) {
				add("GCC006", "no representation for %s", tc)
			}
		}
	}
	for _, s := range t.SupportFiles {
		if _, err := fs.Stat(cat.FS, path.Join(t.Dir, s)); err != nil {
			add("GCC006", "support file %s is missing", s)
		}
	}
	return ds
}

func seen2(rs []Representation, id string) bool {
	for _, r := range rs {
		if r.Type == id {
			return true
		}
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// IDs lists every contract ID in the catalog.
func (cat *Catalog) IDs() []string {
	ids := append(sortedKeys(cat.Types), sortedKeys(cat.Functions)...)
	sort.Strings(ids)
	return ids
}

// Closure returns the function contracts needed by roots, transitively, in
// sorted order.
func (cat *Catalog) Closure(roots []string) ([]string, error) {
	seen := map[string]bool{}
	var visit func(id string) error
	visit = func(id string) error {
		if seen[id] {
			return nil
		}
		f, ok := cat.Functions[id]
		if !ok {
			return fmt.Errorf("unknown runtime function %s", id)
		}
		seen[id] = true
		for _, d := range f.DependsOn {
			if _, isFn := cat.Functions[d.ID]; isFn {
				if err := visit(d.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, r := range roots {
		if err := visit(r); err != nil {
			return nil, err
		}
	}
	return sortedKeys(seen), nil
}

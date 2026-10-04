package golang

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/ir"
)

// LibraryBoundaryError reports a deliberately unsupported public ABI shape.
type LibraryBoundaryError struct{ Message string }

func (e *LibraryBoundaryError) Error() string { return e.Message }

// Library exports deliberately admit immutable value trees, not arbitrary
// source pointers, channels, maps, callbacks or native key handles.
func libraryValue(t *ir.Type, seen map[*ir.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	u := t.U()
	switch u.Kind {
	case ir.KBool, ir.KInt, ir.KString, ir.KFloat:
		return true
	case ir.KSlice, ir.KArray:
		return libraryValue(u.Elem, seen)
	case ir.KStruct:
		for _, f := range u.Fields {
			if !token.IsExported(f.Name) || f.Embedded || !libraryValue(f.Type, seen) {
				return false
			}
		}
		return true
	}
	return false
}
func sourceContext(t *ir.Type) bool { return t.Kind == ir.KOpaque && t.Name == "context.Context" }
func sourceError(t *ir.Type) bool   { return t.Kind == ir.KNamed && t.Name == "error" }

func (e *emitter) library() error {
	e.use("core.task.spawn")
	e.use("std.context.background")
	e.use("std.context.with_cancel")
	e.use("std.context.with_timeout")
	e.use("std.context.err")
	e.p_("type Callbacks = rt.Callbacks\ntype Callback = rt.Callback\ntype LibraryError = rt.LibraryError\n")
	// Public aliases retain native struct fields and exact integer/slice types.
	aliases := map[string]bool{"Callbacks": true, "Callback": true, "LibraryError": true}
	roots := map[string]bool{}
	for _, f := range e.p.Exports {
		roots[f.Pkg] = true
	}
	for _, t := range e.p.Types.All {
		if t.Kind != ir.KNamed || !roots[t.Pkg] || !token.IsExported(t.Obj) {
			continue
		}
		if aliases[t.Obj] {
			return fmt.Errorf("duplicate library public name %s", t.Obj)
		}
		if !libraryValue(t, map[*ir.Type]bool{}) {
			continue
		}
		aliases[t.Obj] = true
		e.p_("type %s = %s\n", t.Obj, e.typ(t))
	}
	publicType := func(t *ir.Type) string {
		if t.Kind == ir.KNamed && roots[t.Pkg] && aliases[t.Obj] {
			return t.Obj
		}
		return e.typ(t)
	}
	e.p_("func libraryReset() {\n")
	for _, g := range e.p.Globals {
		e.p_("%s = *new(%s)\n", g.Sym, e.typ(g.Type))
	}
	e.p_("}\n")
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		if aliases[name] {
			return fmt.Errorf("duplicate library public name %s", name)
		}
		aliases[name] = true
		results := f.Sig.Results
		if f.Sig.Variadic {
			return fmt.Errorf("library export %s: variadic source parameters are unsupported", name)
		}
		for _, t := range f.Sig.Params {
			if !sourceContext(t) && !libraryValue(t, map[*ir.Type]bool{}) {
				return fmt.Errorf("library export %s: unsupported parameter %s", name, t.Name)
			}
		}
		for i, t := range results {
			if !(i == len(results)-1 && sourceError(t)) && !libraryValue(t, map[*ir.Type]bool{}) {
				return fmt.Errorf("library export %s: unsupported result %s", name, t.Name)
			}
		}
		hasError := len(results) > 0 && sourceError(results[len(results)-1])
		n := len(results)
		if hasError {
			n--
		}
		ps := []string{"ctx context.Context"}
		as := []string{}
		for i, t := range f.Sig.Params {
			if sourceContext(t) {
				as = append(as, "sourceCtx")
				continue
			}
			a := fmt.Sprintf("a%d", i)
			ps = append(ps, a+" "+publicType(t))
			as = append(as, a)
		}
		ps = append(ps, "callbacks ...Callbacks")
		rs := []string{}
		for i := 0; i < n; i++ {
			rs = append(rs, fmt.Sprintf("r%d %s", i, publicType(results[i])))
		}
		rs = append(rs, "err error")
		e.p_("func %s(%s) (%s) {\n", name, strings.Join(ps, ", "), strings.Join(rs, ", "))
		for i, t := range f.Sig.Params {
			if !sourceContext(t) {
				e.p_("a%d = rt.Snapshot(a%d)\n", i, i)
			}
		}
		e.p_("var registry Callbacks\nif len(callbacks)>1 { return %s }\nif len(callbacks)==1 { registry=callbacks[0] }\n", libraryErrorReturn(n, "&LibraryError{Kind: \"invalid_callbacks\"}"))
		e.p_("rv, runErr := rt.RunLibrary(ctx, registry, func(sourceCtx rt.Context) rt.Frame {\nlibraryReset()\n")
		// Init and operation are separate roots in one fresh owner; a wrapper frame
		// performs initialization before constructing/calling the export.
		init := e.p.Init.Sym + "()"
		if !e.p.Init.MaySuspend {
			init = "rt.Sync(func() []any { " + e.p.Init.Sym + "(); return nil })"
		}
		call := f.Sym + "(" + strings.Join(as, ", ") + ")"
		if !f.MaySuspend {
			call = "rt.Sync(func() []any { " + syncFrameBody(len(results), call) + " })"
		}
		e.p_("return rt.LibrarySequence(%s, func() rt.Frame { return %s })\n}, libraryReset, func(rv []any) []any {\nowned := make([]any, len(rv))\n", init, call)
		for i := 0; i < n; i++ {
			e.p_("owned[%d] = rt.Snapshot(rv[%d].(%s))\n", i, i, e.typ(results[i]))
		}
		if hasError {
			e.p_("if rv[%d]!=nil { owned[%d]=rt.SnapshotError(rv[%d].(error)) }\n", n, n, n)
		}
		e.p_("return owned\n})\n")
		e.p_("if runErr != nil { return %s }\n", libraryErrorReturn(n, "runErr"))
		// Own every successful result tree before retiring source globals.
		for i := 0; i < n; i++ {
			e.p_("r%d = rv[%d].(%s)\n", i, i, e.typ(results[i]))
		}
		if hasError {
			e.p_("if rv[%d] != nil { err = rv[%d].(error) }\n", n, n)
		}
		e.p_("return\n}\n")
	}
	return nil
}
func libraryErrorReturn(n int, err string) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("r%d", i)
	}
	return strings.Join(append(parts, err), ", ")
}

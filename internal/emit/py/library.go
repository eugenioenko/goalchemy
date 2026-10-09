package py

import (
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"go/token"
	"strings"
)

type LibraryBoundaryError struct{ Message string }

func (e *LibraryBoundaryError) Error() string { return e.Message }
func libraryValue(t *ir.Type, seen map[*ir.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	u := t.U()
	switch u.Kind {
	case ir.KBool, ir.KInt, ir.KString, ir.KFloat:
		return true
	case ir.KArray, ir.KSlice:
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
func (e *emitter) library() (string, error) {
	// These are Go-exported names but Python keywords, so preserving the public
	// name cannot produce a valid Python function declaration in either mode.
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		switch name {
		case "None", "True", "False":
			return "", &LibraryBoundaryError{"reserved Python public function name " + name}
		}
	}

	e.use("core.task.spawn")
	e.use("std.context.with_cancel")
	var b strings.Builder
	seen := map[*ir.Type]bool{}
	var convert func(*ir.Type)
	convert = func(t *ir.Type) {
		u := t.U()
		if seen[u] {
			return
		}
		seen[u] = true
		switch u.Kind {
		case ir.KStruct:
			for _, f := range u.Fields {
				convert(f.Type)
			}
		case ir.KArray, ir.KSlice:
			convert(u.Elem)
		}
		fmt.Fprintf(&b, "def _input_%s(v):\n", e.names.Type(u, ""))
		switch u.Kind {
		case ir.KFloat:
			fmt.Fprintf(&b, "    return rt.library_float(v, %d)\n", u.FloatBits)
		case ir.KBool:
			b.WriteString("    return rt.library_bool(v)\n")
		case ir.KInt:
			fmt.Fprintf(&b, "    return rt.library_int(v, %d, %s)\n", u.Int.Bits(), pyBool(u.Int.Signed()))
		case ir.KString:
			b.WriteString("    return rt.library_string(v)\n")
		case ir.KStruct:
			fmt.Fprintf(&b, "    rt.library_fields(v, %q)\n    r = %s()\n", strings.Join(func() []string {
				var n []string
				for _, f := range u.Fields {
					n = append(n, f.Name)
				}
				return n
			}(), ","), e.class(u))
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "    if %q in v: r.%s = _input_%s(v[%q])\n", f.Name, e.fieldProp(u, i), e.names.Type(f.Type.U(), ""), f.Name)
			}
			b.WriteString("    return r\n")
		case ir.KArray, ir.KSlice:
			if byteElem(u.Elem) {
				b.WriteString("    a = rt.library_bytes(v)\n")
			} else {
				fmt.Fprintf(&b, "    a = None if v is None else [_input_%s(x) for x in rt.library_list(v)]\n", e.names.Type(u.Elem.U(), ""))
			}
			if u.Kind == ir.KArray {
				fmt.Fprintf(&b, "    if a is None or len(a) != %d: raise rt.LibraryFailure('invalid_argument')\n    return a\n", u.Len)
			} else {
				fmt.Fprintf(&b, "    return rt.Slice(a, 0, 0 if a is None else len(a), 0 if a is None else len(a), %s)\n", pyBool(byteElem(u.Elem)))
			}
		}
		fmt.Fprintf(&b, "\ndef _output_%s(v):\n", e.names.Type(u, ""))
		switch u.Kind {
		case ir.KBool, ir.KInt, ir.KString, ir.KFloat:
			b.WriteString("    return v\n")
		case ir.KStruct:
			b.WriteString("    return {\n")
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "        %q: _output_%s(v.%s),\n", f.Name, e.names.Type(f.Type.U(), ""), e.fieldProp(u, i))
			}
			b.WriteString("    }\n")
		case ir.KArray, ir.KSlice:
			expr := "v"
			if u.Kind == ir.KSlice {
				b.WriteString("    if v.a is None: return None\n")
				expr = "v.a[v.o:v.o+v.l]"
			}
			if byteElem(u.Elem) {
				if u.Kind == ir.KSlice {
					expr = "memoryview(v.a)[v.o:v.o+v.l]"
				}
				fmt.Fprintf(&b, "    return bytes(%s)\n", expr)
			} else {
				fmt.Fprintf(&b, "    return [_output_%s(x) for x in %s]\n", e.names.Type(u.Elem.U(), ""), expr)
			}
		}
		b.WriteString("\n")
	}
	for _, f := range e.p.Exports {
		for _, t := range f.Sig.Params {
			if sourceContext(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", &LibraryBoundaryError{"unsupported Python library parameter"}
			}
			convert(t)
		}
		for _, t := range f.Sig.Results {
			if sourceError(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", &LibraryBoundaryError{"unsupported Python library result"}
			}
			convert(t)
		}
	}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] != "" {
			convert(t.Elem)
		}
	}
	b.WriteString("set_log_handler = rt.set_log_handler\nlogging_handler = rt.logging_handler\nLogRecord = rt.LogRecord\n\n")
	b.WriteString("def _source_failure(err):\n    if err is None: return None\n")
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] != "" {
			fmt.Fprintf(&b, "    if err.t is %s: return rt.LibraryFailure('source', _output_%s(err.v))\n", e.tds[t], e.names.Type(t.Elem.U(), ""))
		}
	}
	b.WriteString("    return rt.LibraryFailure('source')\n\ndef _library_reset():\n")
	for _, g := range e.p.Globals {
		slot := e.symbol(g.Sym)
		if g.AddrTaken && !g.Type.IsAggregate() {
			slot += ".v"
		}
		fmt.Fprintf(&b, "    %s = %s\n", slot, e.zero(g.Type))
	}
	b.WriteString("    rt.library_clear_refs()\n\n")
	used := map[string]bool{}
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		if used[name] {
			return "", &LibraryBoundaryError{"duplicate Python export"}
		}
		used[name] = true
		var ps, as, owned []string
		for i, t := range f.Sig.Params {
			if sourceContext(t) {
				as = append(as, "ctx")
				continue
			}
			a := fmt.Sprintf("a%d", i)
			ps = append(ps, a)
			owned = append(owned, a)
			as = append(as, fmt.Sprintf("_input_%s(owned[%d])", e.names.Type(t.U(), ""), len(owned)-1))
		}
		fmt.Fprintf(&b, "def %s(%soptions=None):\n", name, func() string {
			if len(ps) > 0 {
				return strings.Join(ps, ", ") + ", "
			}
			return ""
		}())
		init := e.symbol(e.p.Init.Sym) + "()"
		if !e.p.Init.MaySuspend {
			init = "rt.sync(lambda: (" + init + ", [])[1])"
		}
		call := e.symbol(f.Sym) + "(" + strings.Join(as, ", ") + ")"
		if !f.MaySuspend {
			if len(f.Sig.Results) == 0 {
				call = "rt.sync(lambda: (" + call + ", [])[1])"
			} else if len(f.Sig.Results) == 1 {
				call = "rt.sync(lambda: [" + call + "])"
			} else {
				call = "rt.sync(lambda: list(" + call + "))"
			}
		}
		fmt.Fprintf(&b, "    def factory(ctx, owned):\n        return rt.LibrarySequence(%s, lambda: %s)\n    def output(rv):\n", init, call)
		n := len(f.Sig.Results)
		if n > 0 && sourceError(f.Sig.Results[n-1]) {
			n--
			fmt.Fprintf(&b, "        err = _source_failure(rv[%d])\n        if err is not None: raise err\n", n)
		}
		if n == 0 {
			b.WriteString("        return None\n")
		} else if n == 1 {
			fmt.Fprintf(&b, "        return _output_%s(rv[0])\n", e.names.Type(f.Sig.Results[0].U(), ""))
		} else {
			var vals []string
			for i := 0; i < n; i++ {
				vals = append(vals, fmt.Sprintf("_output_%s(rv[%d])", e.names.Type(f.Sig.Results[i].U(), ""), i))
			}
			fmt.Fprintf(&b, "        return (%s,)\n", strings.Join(vals, ", "))
		}
		fmt.Fprintf(&b, "    return rt.library_submit(factory, output, _library_reset, [%s], options)\n\n", strings.Join(owned, ", "))
	}
	return b.String(), nil
}

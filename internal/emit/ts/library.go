package ts

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
func (e *emitter) library() (string, error) {
	e.use("core.task.spawn")
	e.use("std.context.with_cancel")
	e.use("std.context.err")
	var b strings.Builder
	b.WriteString("export { LibraryError } from './rt/runtime/library.ts';\nexport type { CallOptions, Callback, Settlement } from './rt/runtime/library.ts';\nexport { setLogHandler } from './rt/types/log.ts';\nexport type { LogRecord, LogHandler } from './rt/types/log.ts';\n")
	roots := map[string]bool{}
	for _, f := range e.p.Exports {
		roots[f.Pkg] = true
	}
	names := map[*ir.Type]string{}
	used := map[string]bool{"LibraryError": true, "CallOptions": true, "Callback": true, "LogRecord": true, "LogHandler": true}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KNamed && roots[t.Pkg] && token.IsExported(t.Obj) && libraryValue(t, map[*ir.Type]bool{}) {
			if used[t.Obj] {
				return "", fmt.Errorf("duplicate library public name %s", t.Obj)
			}
			used[t.Obj] = true
			names[t] = t.Obj
		}
	}
	var typ func(*ir.Type) string
	typ = func(t *ir.Type) string {
		if n := names[t]; n != "" {
			return n
		}
		u := t.U()
		switch u.Kind {
		case ir.KFloat:
			return "number"
		case ir.KBool:
			return "boolean"
		case ir.KInt:
			if wide(t) {
				return "bigint"
			}
			return "number"
		case ir.KString:
			return "string"
		case ir.KSlice:
			if byteElem(u.Elem) {
				return "Uint8Array | null"
			}
			return "(" + typ(u.Elem) + ")[] | null"
		case ir.KArray:
			if byteElem(u.Elem) {
				return "Uint8Array"
			}
			return "(" + typ(u.Elem) + ")[]"
		case ir.KStruct:
			var fs []string
			for _, f := range u.Fields {
				fs = append(fs, f.Name+"?: "+typ(f.Type))
			}
			return "{ " + strings.Join(fs, "; ") + " }"
		}
		return "never"
	}
	for _, t := range e.p.Types.All {
		if n := names[t]; n != "" {
			u := t.U()
			if u.Kind == ir.KStruct {
				fmt.Fprintf(&b, "export interface %s {\n", n)
				for _, f := range u.Fields {
					fmt.Fprintf(&b, " %s?: %s;\n", f.Name, typ(f.Type))
				}
				b.WriteString("}\n")
			} else {
				delete(names, t)
				fmt.Fprintf(&b, "export type %s = %s;\n", n, typ(t))
				names[t] = n
			}
		}
	}
	// Generate owner-side recursive conversion for every admitted type.
	seen := map[*ir.Type]bool{}
	var converter func(*ir.Type)
	converter = func(t *ir.Type) {
		u := t.U()
		if seen[u] {
			return
		}
		seen[u] = true
		switch u.Kind {
		case ir.KStruct:
			for _, f := range u.Fields {
				converter(f.Type)
			}
		case ir.KSlice, ir.KArray:
			converter(u.Elem)
		}
		fmt.Fprintf(&b, "function %s(v:any):any {\n", e.names.Type(u, "input$"))
		switch u.Kind {
		case ir.KFloat:
			fmt.Fprintf(&b, "return rt.boundaryFloat(v,%d);\n", u.FloatBits)
		case ir.KBool:
			b.WriteString("return rt.boundaryBool(v);\n")
		case ir.KString:
			b.WriteString("return rt.boundaryString(v);\n")
		case ir.KInt:
			fmt.Fprintf(&b, "return rt.boundaryInt(v,%d,%t);\n", u.Int.Bits(), u.Int.Signed())
		case ir.KStruct:
			fmt.Fprintf(&b, "const o=rt.boundaryObject(v),r=new %s();\n", e.class(u))
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "r.%s=%s(o[%q]);\n", e.fieldProp(u, i), e.names.Type(f.Type.U(), "input$"), f.Name)
			}
			b.WriteString("return r;\n")
		case ir.KSlice:
			if byteElem(u.Elem) {
				b.WriteString("return rt.byteSlice(rt.boundaryBytes(v));\n")
			} else {
				fmt.Fprintf(&b, "const a=rt.boundaryArray(v);return a===null?rt.NIL:rt.fromArray(Array.from({length:a.length},(_,i)=>%s(a[i])));\n", e.names.Type(u.Elem.U(), "input$"))
			}
		case ir.KArray:
			if byteElem(u.Elem) {
				fmt.Fprintf(&b, "const a=rt.boundaryBytes(v);if(a===null||a.length!==%d)rt.invalidBoundary();return a;\n", u.Len)
			} else {
				fmt.Fprintf(&b, "const a=rt.boundaryArray(v);if(a===null||a.length!==%d)rt.invalidBoundary();return Array.from({length:a.length},(_,i)=>%s(a[i]));\n", u.Len, e.names.Type(u.Elem.U(), "input$"))
			}
		}
		b.WriteString("}\n")
		fmt.Fprintf(&b, "function %s(v:any):any {\n", e.names.Type(u, "output$"))
		switch u.Kind {
		case ir.KStruct:
			b.WriteString("return {")
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "%s:%s(v.%s),", f.Name, e.names.Type(f.Type.U(), "output$"), e.fieldProp(u, i))
			}
			b.WriteString("};\n")
		case ir.KSlice:
			if byteElem(u.Elem) {
				b.WriteString("return v.a===null?null:rt.bytes(v);\n")
			} else {
				fmt.Fprintf(&b, "return v.a===null?null:rt.toArray(v).map(x=>%s(x));\n", e.names.Type(u.Elem.U(), "output$"))
			}
		case ir.KArray:
			if byteElem(u.Elem) {
				b.WriteString("return new Uint8Array(v);\n")
			} else {
				fmt.Fprintf(&b, "return v.map(x=>%s(x));\n", e.names.Type(u.Elem.U(), "output$"))
			}
		default:
			b.WriteString("return v;\n")
		}
		b.WriteString("}\n")
	}
	for _, f := range e.p.Exports {
		if f.Sig.Variadic {
			return "", fmt.Errorf("variadic export %s", f.Name)
		}
		for _, t := range f.Sig.Params {
			if !sourceContext(t) {
				if !libraryValue(t, map[*ir.Type]bool{}) {
					return "", fmt.Errorf("unsupported export parameter %s", t.Name)
				}
				converter(t)
			}
		}
		for i, t := range f.Sig.Results {
			if i == len(f.Sig.Results)-1 && sourceError(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", fmt.Errorf("unsupported export result %s", t.Name)
			}
			converter(t)
		}
	}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] != "" {
			converter(t.Elem)
		}
	}
	b.WriteString("function sourceFailure(err:any):rt.LibraryError|null {\nif(err===null)return null;\n")
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] != "" {
			fmt.Fprintf(&b, "if(err.t===%s)return new rt.LibraryError('source',%s(err.v));\n", e.tds[t], e.names.Type(t.Elem.U(), "output$"))
		}
	}
	b.WriteString("return new rt.LibraryError('source');\n}\nfunction libraryReset():void{\n")
	for _, g := range e.p.Globals {
		if g.AddrTaken && !g.Type.IsAggregate() {
			fmt.Fprintf(&b, "%s.v=%s;\n", e.symbol(g.Sym), e.zero(g.Type))
		} else {
			fmt.Fprintf(&b, "%s=%s;\n", e.symbol(g.Sym), e.zero(g.Type))
		}
	}
	b.WriteString("}\n")
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		if used[name] {
			return "", fmt.Errorf("duplicate export name %s", name)
		}
		used[name] = true
		var ps, as []string
		for i, t := range f.Sig.Params {
			if sourceContext(t) {
				as = append(as, "ctx")
				continue
			}
			a := fmt.Sprintf("a%d", i)
			ps = append(ps, a+": "+typ(t))
			as = append(as, a)
		}
		ps = append(ps, "options: rt.CallOptions = {}")
		n := len(f.Sig.Results)
		hasErr := n > 0 && sourceError(f.Sig.Results[n-1])
		if hasErr {
			n--
		}
		result := "void"
		if n == 1 {
			result = typ(f.Sig.Results[0])
		} else if n > 1 {
			var rs []string
			for i := 0; i < n; i++ {
				rs = append(rs, typ(f.Sig.Results[i]))
			}
			result = "[" + strings.Join(rs, ",") + "]"
		}
		fmt.Fprintf(&b, "export async function %s(%s):Promise<%s>{\noptions=rt.snapshotOptions(options);\n", name, strings.Join(ps, ","), result)
		for i, t := range f.Sig.Params {
			if !sourceContext(t) {
				fmt.Fprintf(&b, "a%d=%s(a%d);\n", i, e.names.Type(t.U(), "input$"), i)
			}
		}
		init := e.symbol(e.p.Init.Sym) + "()"
		if !e.p.Init.MaySuspend {
			init = "rt.sync(()=>{ " + e.symbol(e.p.Init.Sym) + "();return [];})"
		}
		call := e.symbol(f.Sym) + "(" + strings.Join(as, ",") + ")"
		if !f.MaySuspend {
			body := call
			if len(f.Sig.Results) == 0 {
				body = call + ";return []"
			} else if len(f.Sig.Results) == 1 {
				body = "return [" + call + "]"
			} else {
				body = "return " + call
			}
			call = "rt.sync(()=>{ " + body + ";})"
		}
		fmt.Fprintf(&b, "const owned=await rt.runLibrary(options,ctx=>rt.librarySequence(%s,()=>%s),libraryReset,rv=>{\nreturn [", init, call)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, "%s(rv[%d]),", e.names.Type(f.Sig.Results[i].U(), "output$"), i)
		}
		if hasErr {
			fmt.Fprintf(&b, "sourceFailure(rv[%d]),", n)
		}
		b.WriteString("];});\n")
		if hasErr {
			fmt.Fprintf(&b, "if(owned[%d]!==null)throw owned[%d];\n", n, n)
		}
		if n == 1 {
			b.WriteString("return owned[0] as any;\n")
		} else if n > 1 {
			fmt.Fprintf(&b, "return owned.slice(0,%d) as any;\n", n)
		}
		b.WriteString("}\n")
	}
	return b.String(), nil
}

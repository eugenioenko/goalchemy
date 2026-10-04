package java

import (
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"go/token"
	"strings"
)

// LibraryBoundaryError identifies deliberately unsupported public value shapes.
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
func publicArray(element, length string) string {
	dims := ""
	for strings.HasSuffix(element, "[]") {
		element = strings.TrimSuffix(element, "[]")
		dims += "[]"
	}
	return "new " + element + "[" + length + "]" + dims
}

func (e *emitter) library() (string, error) {
	e.use("core.task.spawn")
	e.use("std.context.err")
	e.use("std.context.with_cancel")
	var b strings.Builder
	names := map[*ir.Type]string{}
	roots := map[string]bool{}
	used := map[string]bool{}
	reserved := map[string]bool{}
	for _, name := range strings.Fields("Generated Library Callback Native TaskSpawn Program Box Slice Ref Cell Fn TypeDesc Bounds Ints Floats Out Utf8 Desc GoMap GoPanic FatalPanic Panics Channel StdContextErr StdContextWithCancel EnvFn Object String Long Boolean Float Double Void Integer Math System Throwable RuntimeException") {
		reserved[name] = true
	}
	for _, symbol := range e.symbols {
		reserved[strings.Split(symbol, ".")[0]] = true
	}
	for _, f := range e.p.Exports {
		roots[f.Pkg] = true
	}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KNamed && roots[t.Pkg] && token.IsExported(t.Obj) && t.U().Kind == ir.KStruct && libraryValue(t, map[*ir.Type]bool{}) {
			if reserved[t.Obj] {
				return "", fmt.Errorf("reserved Java public type name %s", t.Obj)
			}
			if used[t.Obj] {
				return "", fmt.Errorf("duplicate public name %s", t.Obj)
			}
			used[t.Obj] = true
			names[t.U()] = t.Obj
		}
	}
	var typ func(*ir.Type) string
	typ = func(t *ir.Type) string {
		u := t.U()
		switch u.Kind {
		case ir.KBool:
			return "boolean"
		case ir.KFloat:
			if u.FloatBits == 32 {
				return "float"
			}
			return "double"
		case ir.KInt:
			return "long"
		case ir.KString:
			return "String"
		case ir.KStruct:
			if n := names[u]; n != "" {
				return n
			}
			n := fmt.Sprintf("Value%d", u.ID)
			names[u] = n
			return n
		case ir.KArray, ir.KSlice:
			if byteElem(u.Elem) {
				return "byte[]"
			}
			return typ(u.Elem) + "[]"
		}
		return "Object"
	}
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
			n := typ(t)
			for _, f := range u.Fields {
				convert(f.Type)
			}
			fmt.Fprintf(&b, " public static final class %s {\n", n)
			for _, f := range u.Fields {
				z := ""
				if f.Type.U().Kind == ir.KString {
					z = " = \"\""
				}
				fmt.Fprintf(&b, "  public %s %s%s;\n", typ(f.Type), f.Name, z)
			}
			b.WriteString(" }\n")
		case ir.KArray, ir.KSlice:
			convert(u.Elem)
		}
		id := u.ID
		fmt.Fprintf(&b, " private static %s snapshot$%d(%s v,Library.CopyContext copy) {\n", typ(t), id, typ(t))
		aggregate := u.Kind == ir.KStruct || u.Kind == ir.KArray || u.Kind == ir.KSlice
		if aggregate {
			b.WriteString("Object original=v;copy.enter(original);try {\n")
		}
		switch u.Kind {
		case ir.KBool, ir.KFloat:
			b.WriteString("return v;\n")
		case ir.KInt:
			fmt.Fprintf(&b, "return Library.integer(v,%d,%t);\n", u.Int.Bits(), u.Int.Signed())
		case ir.KString:
			b.WriteString("return Library.binaryString(v);\n")
		case ir.KStruct:
			fmt.Fprintf(&b, "if(v==null)v=new %s(); %s r=new %s();\n", typ(t), typ(t), typ(t))
			for _, f := range u.Fields {
				fmt.Fprintf(&b, "r.%s=snapshot$%d(v.%s,copy);\n", f.Name, f.Type.U().ID, f.Name)
			}
			b.WriteString("return r;\n")
		case ir.KArray, ir.KSlice:
			if u.Kind == ir.KArray {
				fmt.Fprintf(&b, "if(v==null||v.length!=%d)throw Library.invalid();\n", u.Len)
			}
			if byteElem(u.Elem) {
				b.WriteString("return Library.bytes(v);\n")
			} else {
				b.WriteString("if(v==null)return null;Library.length(v.length);\n")
				fmt.Fprintf(&b, "%s r=%s;for(int i=0;i<r.length;i++)r[i]=snapshot$%d(v[i],copy);return r;\n", typ(t), publicArray(typ(u.Elem), "v.length"), u.Elem.U().ID)
			}
		}
		if aggregate {
			b.WriteString("} finally {copy.leave(original); }\n")
		}
		b.WriteString(" }\n")
		fmt.Fprintf(&b, " private static %s input$%d(%s v) {\n", e.jt(t), id, typ(t))
		switch u.Kind {
		case ir.KBool, ir.KFloat:
			b.WriteString("return v;\n")
		case ir.KInt:
			fmt.Fprintf(&b, "return Library.integer(v,%d,%t);\n", u.Int.Bits(), u.Int.Signed())
		case ir.KString:
			b.WriteString("return Library.binaryString(v);\n")
		case ir.KStruct:
			fmt.Fprintf(&b, "if(v==null)v=new %s(); %s r=new %s();\n", typ(t), e.jt(t), e.jt(t))
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "r.%s=input$%d(v.%s);\n", fieldProp(u, i), f.Type.U().ID, f.Name)
			}
			b.WriteString("return r;\n")
		case ir.KArray, ir.KSlice:
			if byteElem(u.Elem) {
				if u.Kind == ir.KArray {
					fmt.Fprintf(&b, "if(v==null||v.length!=%d)throw Library.invalid(); return Library.bytes(v);\n", u.Len)
				} else {
					b.WriteString("byte[] a=Library.bytes(v); return a==null?Slice.BYTE_NIL:new Slice(a,0,a.length,a.length);\n")
				}
			} else {
				if u.Kind == ir.KArray {
					fmt.Fprintf(&b, "if(v==null||v.length!=%d)throw Library.invalid();\n", u.Len)
				} else {
					b.WriteString("if(v==null)return Slice.NIL;\n")
				}
				b.WriteString("Library.length(v.length);Object[] a=new Object[v.length];\n")
				fmt.Fprintf(&b, "for(int i=0;i<a.length;i++)a[i]=input$%d(v[i]);\n", u.Elem.U().ID)
				if u.Kind == ir.KArray {
					b.WriteString("return a;\n")
				} else {
					b.WriteString("return new Slice(a,0,a.length,a.length);\n")
				}
			}
		}
		b.WriteString(" }\n")
		fmt.Fprintf(&b, " private static %s output$%d(%s v) {\n", typ(t), id, e.jt(t))
		switch u.Kind {
		case ir.KStruct:
			fmt.Fprintf(&b, "%s r=new %s();\n", typ(t), typ(t))
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "r.%s=output$%d(v.%s);\n", f.Name, f.Type.U().ID, fieldProp(u, i))
			}
			b.WriteString("return r;\n")
		case ir.KArray, ir.KSlice:
			if byteElem(u.Elem) {
				if u.Kind == ir.KArray {
					b.WriteString("return v.clone();\n")
				} else {
					b.WriteString("return Native.bytesNullable(v);\n")
				}
			} else {
				length := "v.length"
				item := "v[i]"
				if u.Kind == ir.KSlice {
					b.WriteString("if(v.a==null)return null;\n")
					length = "v.l"
					item = "v.get(i)"
				}
				fmt.Fprintf(&b, "%s a=%s;\n", typ(t), publicArray(typ(u.Elem), length))
				fmt.Fprintf(&b, "for(int i=0;i<a.length;i++)a[i]=output$%d(%s);return a;\n", u.Elem.U().ID, e.cast(u.Elem, item))
			}
		default:
			b.WriteString("return v;\n")
		}
		b.WriteString(" }\n")
		fmt.Fprintf(&b, " private static Object wire$%d(%s v) {\n", id, typ(t))
		switch u.Kind {
		case ir.KStruct:
			b.WriteString("java.util.Map<String,Object> r=new java.util.LinkedHashMap<>();\n")
			for _, f := range u.Fields {
				fmt.Fprintf(&b, "r.put(%q,wire$%d(v.%s));\n", f.Name, f.Type.U().ID, f.Name)
			}
			b.WriteString("return r;\n")
		case ir.KArray, ir.KSlice:
			if byteElem(u.Elem) {
				b.WriteString("return v==null?null:v.clone();\n")
			} else {
				fmt.Fprintf(&b, "if(v==null)return null;Object[] a=new Object[v.length];for(int i=0;i<a.length;i++)a[i]=wire$%d(v[i]);return a;\n", u.Elem.U().ID)
			}
		default:
			b.WriteString("return v;\n")
		}
		b.WriteString(" }\n")
	}
	for _, f := range e.p.Exports {
		if f.Sig.Variadic {
			return "", fmt.Errorf("variadic export %s", f.Name)
		}
		for _, t := range f.Sig.Params {
			if !sourceContext(t) {
				if !libraryValue(t, map[*ir.Type]bool{}) {
					return "", fmt.Errorf("unsupported parameter %s", t.Name)
				}
				convert(t)
			}
		}
		for i, t := range f.Sig.Results {
			if i == len(f.Sig.Results)-1 && sourceError(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", fmt.Errorf("unsupported result %s", t.Name)
			}
			convert(t)
		}
	}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] != "" {
			convert(t.Elem)
		}
	}
	b.WriteString(" private static Library.Failure sourceFailure(Box err) {\nif(err==null)return null;\n")
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] != "" {
			fmt.Fprintf(&b, "if(err.t==%s) { %s v=output$%d((%s)err.v); java.util.Map<String,Object> fields=new java.util.LinkedHashMap<>();\n", e.tds[t], typ(t.Elem), t.Elem.U().ID, e.jt(t.Elem))
			for _, f := range t.Elem.U().Fields {
				fmt.Fprintf(&b, "fields.put(%q,wire$%d(v.%s));\n", f.Name, f.Type.U().ID, f.Name)
			}
			b.WriteString("return new Library.Failure(\"source\",fields); }\n")
		}
	}
	b.WriteString("return new Library.Failure(\"source\");\n }\n private static void libraryReset() {\n")
	for _, g := range e.p.Globals {
		if g.AddrTaken && !g.Type.IsAggregate() {
			fmt.Fprintf(&b, "%s.v=%s;\n", g.Sym, e.zero(g.Type))
		} else {
			fmt.Fprintf(&b, "%s=%s;\n", g.Sym, e.zero(g.Type))
		}
	}
	b.WriteString("Program.clearFieldRefs();\n }\n")
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		if used[name] {
			return "", fmt.Errorf("duplicate export %s", name)
		}
		used[name] = true
		n := len(f.Sig.Results)
		hasErr := n > 0 && sourceError(f.Sig.Results[n-1])
		if hasErr {
			n--
		}
		result := "Void"
		if n == 1 {
			result = typ(f.Sig.Results[0])
			if result == "long" {
				result = "Long"
			}
			if result == "boolean" {
				result = "Boolean"
			}
			if result == "float" {
				result = "Float"
			}
			if result == "double" {
				result = "Double"
			}
		} else if n > 1 {
			result = name + "Result"
			if used[result] || reserved[result] {
				return "", fmt.Errorf("duplicate Java result type %s", result)
			}
			used[result] = true
			fmt.Fprintf(&b, "public record %s(", result)
			for i := 0; i < n; i++ {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, "%s value%d", typ(f.Sig.Results[i]), i)
			}
			b.WriteString(") {}\n")
		}
		var ps, as []string
		for i, t := range f.Sig.Params {
			if sourceContext(t) {
				as = append(as, "ctx")
				continue
			}
			a := fmt.Sprintf("a%d", i)
			ps = append(ps, typ(t)+" "+a)
			as = append(as, fmt.Sprintf("input$%d(owned%s)", t.U().ID, a))
		}
		ps = append(ps, "Library.Options options")
		fmt.Fprintf(&b, " public static Library.Operation<%s> %s(%s) {\ntry {\nLibrary.CopyContext copy=new Library.CopyContext();\n", result, name, strings.Join(ps, ","))
		for i, t := range f.Sig.Params {
			if !sourceContext(t) {
				fmt.Fprintf(&b, "final %s owneda%d=snapshot$%d(a%d,copy);\n", typ(t), i, t.U().ID, i)
			}
		}
		init := e.p.Init.Sym + "()"
		if !e.p.Init.MaySuspend {
			init = "TaskSpawn.sync(() -> { " + e.p.Init.Sym + "();return new Object[0];})"
		}
		call := f.Sym + "(" + strings.Join(as, ",") + ")"
		if !f.MaySuspend {
			body := call + ";return new Object[0]"
			if len(f.Sig.Results) == 1 {
				body = "return new Object[] {" + call + "}"
			} else if len(f.Sig.Results) > 1 {
				body = "return " + call
			}
			call = "TaskSpawn.sync(() -> { " + body + "; })"
		}
		fmt.Fprintf(&b, "return Library.submit(options,ctx -> Library.sequence(%s,()->%s),Generated::libraryReset,rv -> {\n", init, call)
		if hasErr {
			fmt.Fprintf(&b, "Library.Failure err=sourceFailure((Box)rv[%d]);if(err!=null)throw err;\n", n)
		}
		if n == 1 {
			fmt.Fprintf(&b, "return output$%d(%s);\n", f.Sig.Results[0].U().ID, e.cast(f.Sig.Results[0], "rv[0]"))
		} else if n > 1 {
			fmt.Fprintf(&b, "return new %s(", result)
			for i := 0; i < n; i++ {
				if i > 0 {
					b.WriteString(",")
				}
				fmt.Fprintf(&b, "output$%d(%s)", f.Sig.Results[i].U().ID, e.cast(f.Sig.Results[i], fmt.Sprintf("rv[%d]", i)))
			}
			b.WriteString(");\n")
		} else {
			b.WriteString("return null;\n")
		}
		b.WriteString("});\n} catch(Library.Failure e) { return Library.failed(e); }\n }\n")
	}
	return b.String(), nil
}

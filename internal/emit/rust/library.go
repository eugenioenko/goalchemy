package rust

import (
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"go/token"
	"sort"
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
	e.use("core.task.spawn")
	e.use("std.context.with_cancel")
	e.use("lib.crypto.close")
	names := map[*ir.Type]string{}
	roots := map[string]bool{}
	for _, f := range e.p.Exports {
		roots[f.Pkg] = true
	}
	used := map[string]bool{}
	reserved := map[string]bool{}
	for _, name := range strings.Fields("V H Key Code Func TypeDesc PanicObj GoPanicPayload ErrorKind LibraryError Cancellation ProviderRequest ProviderError Provider CallOptions Operation HostFault HostFatal SourceStackFatal HostWire HostRecord HostMailbox HostToken HostBoundary HostPending Step Results Frame Task Timer Sched Obj Heap Deferred FrData Fr TempRoots KeyIndex Entry GoMap MapIter EntryReservation SourceGuard Waiter WaiterData Chan Mutex WaitGroup ContextHook Context ContextHookHandle BlockedPayload HostError Vec Rc Cell RefCell Option Result String Some None Ok Err") {
		reserved[name] = true
	}
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		if reserved[name] {
			return "", &LibraryBoundaryError{fmt.Sprintf("reserved Rust public function name %s", name)}
		}
		if used[name] {
			return "", &LibraryBoundaryError{fmt.Sprintf("duplicate Rust public name %s", name)}
		}
		used[name] = true
	}
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KNamed && roots[t.Pkg] && t.U().Kind == ir.KStruct && token.IsExported(t.Obj) && libraryValue(t, map[*ir.Type]bool{}) {
			n := t.Obj
			if reserved[n] {
				return "", &LibraryBoundaryError{fmt.Sprintf("reserved Rust public type name %s", n)}
			}
			if used[n] {
				return "", &LibraryBoundaryError{fmt.Sprintf("duplicate Rust public type name %s", n)}
			}
			used[n] = true
			names[t.U()] = n
		}
	}
	var typ func(*ir.Type) string
	typ = func(t *ir.Type) string {
		u := t.U()
		switch u.Kind {
		case ir.KBool:
			return "bool"
		case ir.KFloat:
			return fmt.Sprintf("f%d", u.FloatBits)
		case ir.KInt:
			return intKind(u)
		case ir.KString:
			return "Vec<u8>"
		case ir.KArray, ir.KSlice:
			return "Vec<" + typ(u.Elem) + ">"
		case ir.KStruct:
			if names[u] == "" {
				n := fmt.Sprintf("Value%d", u.ID)
				for used[n] || reserved[n] {
					n += "_"
				}
				used[n] = true
				names[u] = n
			}
			return names[u]
		}
		return "()"
	}
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
			typ(u)
			for _, f := range u.Fields {
				convert(f.Type)
			}
		case ir.KArray, ir.KSlice:
			convert(u.Elem)
		}
		if u.Kind == ir.KStruct {
			fmt.Fprintf(&b, "#[derive(Clone, Debug, Default, PartialEq)]\npub struct %s {\n", typ(u))
			for _, f := range u.Fields {
				fmt.Fprintf(&b, "pub %s: %s,\n", f.Name, typ(f.Type))
			}
			b.WriteString("}\n")
		}
		fmt.Fprintf(&b, "fn input_%s(v: %s) -> V {\n", e.names.Type(u, ""), typ(u))
		switch u.Kind {
		case ir.KFloat:
			b.WriteString("V::Float(v as f64)\n")
		case ir.KBool:
			b.WriteString("V::Bool(v)\n")
		case ir.KInt:
			b.WriteString("V::Int(v as i64)\n")
		case ir.KString:
			b.WriteString("s(&v)\n")
		case ir.KStruct:
			var fields []string
			for _, f := range u.Fields {
				fields = append(fields, fmt.Sprintf("input_%s(v.%s)", e.names.Type(f.Type.U(), ""), f.Name))
			}
			fmt.Fprintf(&b, "vals(vec![%s])\n", strings.Join(fields, ","))
		case ir.KArray, ir.KSlice:
			b.WriteString("let n=library_length(v.len());\n")
			if u.Kind == ir.KArray {
				fmt.Fprintf(&b, "if v.len()!=%d { library_invalid(\"invalid native array length\"); }\n", u.Len)
			}
			if byteElem(u.Elem) {
				if u.Kind == ir.KArray {
					b.WriteString("byte_array(v)\n")
				} else {
					b.WriteString("library_bytes(v)\n")
				}
			} else {
				fmt.Fprintf(&b, "let h=vals(v.into_iter().map(input_%s).collect()).h();\n", e.names.Type(u.Elem.U(), ""))
				if u.Kind == ir.KArray {
					b.WriteString("V::Obj(h)\n")
				} else {
					b.WriteString("V::Slice(h,0,n,n)\n")
				}
			}
		}
		b.WriteString("}\n")
		fmt.Fprintf(&b, "fn output_%s(v: V) -> %s {\n", e.names.Type(u, ""), typ(u))
		switch u.Kind {
		case ir.KFloat:
			fmt.Fprintf(&b, "v.f() as f%d\n", u.FloatBits)
		case ir.KBool:
			b.WriteString("v.b()\n")
		case ir.KInt:
			fmt.Fprintf(&b, "v.i() as %s\n", typ(u))
		case ir.KString:
			b.WriteString("v.bytes().to_vec()\n")
		case ir.KStruct:
			fmt.Fprintf(&b, "%s {\n", typ(u))
			for i, f := range u.Fields {
				fmt.Fprintf(&b, "%s:output_%s(fld(&v,%d)),\n", f.Name, e.names.Type(f.Type.U(), ""), i)
			}
			b.WriteString("}\n")
		case ir.KArray, ir.KSlice:
			if byteElem(u.Elem) {
				if u.Kind == ir.KArray {
					b.WriteString("byte_snapshot(v.h(),0,vals_len(v.h()))\n")
				} else {
					b.WriteString("library_byte_output(&v)\n")
				}
			} else {
				if u.Kind == ir.KArray {
					fmt.Fprintf(&b, "(0..%d).map(|i|output_%s(slot(v.h(),i))).collect()\n", u.Len, e.names.Type(u.Elem.U(), ""))
				} else {
					fmt.Fprintf(&b, "let(h,o,n,_,_)=slice_parts(&v);(0..n as usize).map(|i|output_%s(slot(h,o as usize+i))).collect()\n", e.names.Type(u.Elem.U(), ""))
				}
			}
		}
		b.WriteString("}\n")
	}
	for _, f := range e.p.Exports {
		for _, t := range f.Sig.Params {
			if sourceContext(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", &LibraryBoundaryError{"unsupported Rust library parameter"}
			}
			convert(t)
		}
		for _, t := range f.Sig.Results {
			if sourceError(t) {
				continue
			}
			if !libraryValue(t, map[*ir.Type]bool{}) {
				return "", &LibraryBoundaryError{"unsupported Rust library result"}
			}
			convert(t)
		}
	}
	b.WriteString("pub use rt::{Operation, CallOptions, LibraryError, ErrorKind, Cancellation, Provider, ProviderRequest, ProviderError, HostWire};\nfn source_failure(v: &V) -> LibraryError {\nif veq(v,&context_canceled()) {return LibraryError::new(ErrorKind::Canceled,\"context canceled\");}\nif veq(v,&context_deadline_exceeded()) {return LibraryError::new(ErrorKind::DeadlineExceeded,\"context deadline exceeded\");}\n")
	var errorTypes []*ir.Type
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && libraryValue(t.Elem, map[*ir.Type]bool{}) && e.tds[t] {
			errorTypes = append(errorTypes, t)
		}
	}
	sort.Slice(errorTypes, func(i, j int) bool { return errorTypes[i].ID < errorTypes[j].ID })
	for _, t := range errorTypes {
		fmt.Fprintf(&b, "if is_type(v, %s) { let x=unboxed(v); let mut err=LibraryError::from_bytes(ErrorKind::Source,format_panic_value(v));\n", e.td(t))
		for i, f := range t.Elem.U().Fields {
			switch f.Type.U().Kind {
			case ir.KString:
				fmt.Fprintf(&b, "err.fields.insert(%q.into(), HostWire::Bytes(fld(&x,%d).bytes().to_vec()));\n", f.Name, i)
			case ir.KInt:
				fmt.Fprintf(&b, "err.fields.insert(%q.into(), HostWire::Int(fld(&x,%d).i()));\n", f.Name, i)
			case ir.KSlice:
				if f.Type.U().Elem.U().Kind == ir.KString {
					fmt.Fprintf(&b, "err.fields.insert(%q.into(), HostWire::List(library_strings(fld(&x,%d)).into_iter().map(HostWire::Bytes).collect()));\n", f.Name, i)
				}
			}
		}
		b.WriteString("return err.normalize_cause(); }\n")
	}
	b.WriteString("LibraryError::from_bytes(ErrorKind::Source,format_panic_value(v))\n}\n")
	for _, f := range e.p.Exports {
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		var ps, args []string
		for i, t := range f.Sig.Params {
			if sourceContext(t) {
				args = append(args, "ctx.clone()")
				continue
			}
			a := fmt.Sprintf("a%d", i)
			ps = append(ps, a+": "+typ(t))
			args = append(args, fmt.Sprintf("input_%s(%s)", e.names.Type(t.U(), ""), a))
		}
		n := len(f.Sig.Results)
		hasError := n > 0 && sourceError(f.Sig.Results[n-1])
		if hasError {
			n--
		}
		var rs, outs []string
		for i := 0; i < n; i++ {
			rs = append(rs, typ(f.Sig.Results[i]))
			outs = append(outs, fmt.Sprintf("output_%s(rv[%d].clone())", e.names.Type(f.Sig.Results[i].U(), ""), i))
		}
		result := "()"
		out := "()"
		if n == 1 {
			result = rs[0]
			out = outs[0]
		} else if n > 1 {
			result = "(" + strings.Join(rs, ",") + ")"
			out = "(" + strings.Join(outs, ",") + ")"
		}
		ps = append(ps, "options: CallOptions")
		fmt.Fprintf(&b, "pub fn %s(%s) -> Operation<%s> {\nOperation::submit(options, move |options| {\nlibrary_owner(%d, init_zero_globals, options, |ctx| {\n", name, strings.Join(ps, ","), result, len(e.p.Globals))
		init := fmt.Sprintf("%s()", e.symbol(e.p.Init.Sym, "f_"))
		if !e.p.Init.MaySuspend {
			init = fmt.Sprintf("sync_frame(func(-1,%s,vec![]),vec![],0)", e.symbol(e.p.Init.Sym, "w_"))
		}
		call := fmt.Sprintf("%s(%s)", e.symbol(f.Sym, "f_"), strings.Join(args, ","))
		if !f.MaySuspend {
			call = fmt.Sprintf("sync_frame(func(-1,%s,vec![]),vec![%s],%d)", e.symbol(f.Sym, "w_"), strings.Join(args, ","), len(f.Sig.Results))
		}
		fmt.Fprintf(&b, "library_sequence(%s, move || %s)\n}, |rv| {\n", init, call)
		if hasError {
			fmt.Fprintf(&b, "if !rv[%d].is_nil() {return Err(source_failure(&rv[%d]));}\n", n, n)
		}
		fmt.Fprintf(&b, "Ok(%s)\n})\n})\n}\n", out)
	}
	return b.String(), nil
}

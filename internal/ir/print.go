package ir

import (
	"fmt"
	"io"
	"strings"
)

// Dump writes a human-readable rendering of the program.
func Dump(w io.Writer, p *Program) {
	for _, g := range p.Globals {
		fmt.Fprintf(w, "global %s %s\n", g.Sym, g.Type)
	}
	for _, f := range p.Funcs {
		DumpFunc(w, f)
	}
}

func DumpFunc(w io.Writer, f *Func) {
	var ps []string
	for _, p := range f.Params {
		ps = append(ps, localName(p)+" "+p.Type.String())
	}
	var rs []string
	for _, r := range f.Results {
		rs = append(rs, localName(r)+" "+r.Type.String())
	}
	var env []string
	for _, e := range f.Env {
		env = append(env, localName(e))
	}
	fmt.Fprintf(w, "func %s(%s) (%s)", f.Sym, strings.Join(ps, ", "), strings.Join(rs, ", "))
	if len(env) > 0 {
		fmt.Fprintf(w, " env(%s)", strings.Join(env, ", "))
	}
	fmt.Fprintln(w)
	for _, b := range f.Blocks {
		fmt.Fprintf(w, "  b%d: ; %s\n", b.ID, b.Comment)
		for _, in := range b.Instrs {
			fmt.Fprintf(w, "    %s\n", InstrString(in))
		}
		fmt.Fprintf(w, "    %s\n", InstrString(b.Term))
	}
}

func localName(l *Local) string {
	n := l.Name
	if n == "" {
		n = "t"
	}
	s := fmt.Sprintf("%s#%d", n, l.ID)
	if l.Boxed {
		s += "^"
	}
	return s
}

func ValueString(v Value) string {
	switch v := v.(type) {
	case *Local:
		return localName(v)
	case *Const:
		if v.Nil {
			return "nil:" + v.Type.String()
		}
		return v.Val.ExactString() + ":" + v.Type.String()
	case *FuncRef:
		return "&" + v.Func.Sym
	case nil:
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
}

func PlaceString(p *Place) string {
	var s string
	switch r := p.Root.(type) {
	case LocalRoot:
		s = localName(r.Local)
	case GlobalRoot:
		s = r.Global.Sym
	case DerefRoot:
		s = "*" + ValueString(r.Ptr)
	case SliceRoot:
		s = ValueString(r.Slice) + "[" + ValueString(r.Index) + "]"
	case ValueRoot:
		s = "(" + ValueString(r.Value) + ")"
	}
	for _, pr := range p.Path {
		if pr.Index != nil {
			s += "[" + ValueString(pr.Index) + "]"
		} else {
			s += fmt.Sprintf(".f%d", pr.Field)
		}
	}
	return s
}

func vals(vs []Value) string {
	var out []string
	for _, v := range vs {
		out = append(out, ValueString(v))
	}
	return strings.Join(out, ", ")
}

func InstrString(in any) string {
	switch i := in.(type) {
	case *DeclVar:
		if i.Init == nil {
			return "var " + localName(i.L) + " " + i.L.Type.String()
		}
		return "var " + localName(i.L) + " = " + ValueString(i.Init)
	case *Assign:
		return localName(i.Dst) + " := " + ValueString(i.Src)
	case *Zero:
		return localName(i.Dst) + " = zero " + i.Dst.Type.String()
	case *Load:
		return localName(i.Dst) + " = load " + PlaceString(i.Place)
	case *Store:
		return "store " + PlaceString(i.Place) + " = " + ValueString(i.V)
	case *AddrOf:
		return localName(i.Dst) + " = &" + PlaceString(i.Place)
	case *New:
		return localName(i.Dst) + " = new " + i.Dst.Type.String()
	case *UnOp:
		return fmt.Sprintf("%s = %s %s", localName(i.Dst), [...]string{"?", "-", "!", "^"}[i.Op], ValueString(i.X))
	case *BinOp:
		return fmt.Sprintf("%s = %s %s %s", localName(i.Dst), ValueString(i.X), i.Op, ValueString(i.Y))
	case *Convert:
		return fmt.Sprintf("%s = convert(%d) %s to %s", localName(i.Dst), i.Kind, ValueString(i.X), i.Dst.Type)
	case *MakeInterface:
		return fmt.Sprintf("%s = box %s as %s", localName(i.Dst), ValueString(i.X), i.Dst.Type)
	case *TypeAssert:
		s := localName(i.Dst)
		if i.Ok != nil {
			s += ", " + localName(i.Ok)
		}
		return s + " = " + ValueString(i.X) + ".(" + i.T.String() + ")"
	case *Call:
		var ds []string
		for _, d := range i.Dsts {
			ds = append(ds, localName(d))
		}
		var callee string
		switch i.Kind {
		case CallStatic:
			callee = i.Func.Sym
		case CallValue:
			callee = ValueString(i.Fn)
		case CallInterface:
			callee = ValueString(i.Recv) + "." + i.Method
		case CallExtern:
			callee = "extern " + i.Extern.Contract
		}
		return strings.Join(ds, ", ") + " = call " + callee + "(" + vals(i.Args) + ")"
	case *MakeClosure:
		var env []string
		for _, e := range i.Env {
			env = append(env, localName(e))
		}
		return localName(i.Dst) + " = closure " + i.Func.Sym + "[" + strings.Join(env, ", ") + "]"
	case *MakeBound:
		return localName(i.Dst) + " = bound " + i.Func.Sym + "(" + ValueString(i.Recv) + ")"
	case *MakeIfaceBound:
		return localName(i.Dst) + " = ibound " + ValueString(i.Recv) + "." + i.Method
	case *Len:
		return localName(i.Dst) + " = len " + ValueString(i.X)
	case *Cap:
		return localName(i.Dst) + " = cap " + ValueString(i.X)
	case *MakeSlice:
		return fmt.Sprintf("%s = make %s(%s, %s)", localName(i.Dst), i.Dst.Type, ValueString(i.Len), ValueString(i.Cap))
	case *MakeMap:
		return localName(i.Dst) + " = make " + i.Dst.Type.String()
	case *Append:
		if i.Spread != nil {
			return localName(i.Dst) + " = append " + ValueString(i.S) + ", " + ValueString(i.Spread) + "..."
		}
		return localName(i.Dst) + " = append " + ValueString(i.S) + ", " + vals(i.Elems)
	case *Copy:
		return localName(i.N) + " = copy " + ValueString(i.Dst) + ", " + ValueString(i.Src)
	case *IndexString:
		return localName(i.Dst) + " = " + ValueString(i.S) + "[" + ValueString(i.I) + "]"
	case *SliceOp:
		return localName(i.Dst) + " = " + ValueString(i.X) + "[" + ValueString(i.Lo) + ":" + ValueString(i.Hi) + ":" + ValueString(i.Max) + "]"
	case *MapLookup:
		s := localName(i.Dst)
		if i.Ok != nil {
			s += ", " + localName(i.Ok)
		}
		return s + " = " + ValueString(i.M) + "[" + ValueString(i.K) + "]"
	case *MapStore:
		return ValueString(i.M) + "[" + ValueString(i.K) + "] = " + ValueString(i.V)
	case *MapDelete:
		return "delete " + ValueString(i.M) + ", " + ValueString(i.K)
	case *Clear:
		return "clear " + ValueString(i.X)
	case *DecodeRune:
		return localName(i.Rune) + ", " + localName(i.Width) + " = decoderune " + ValueString(i.S) + ", " + ValueString(i.I)
	case *MapIterInit:
		return localName(i.Iter) + " = mapiter " + ValueString(i.M)
	case *MapIterNext:
		k, v := "_", "_"
		if i.Key != nil {
			k = localName(i.Key)
		}
		if i.Val != nil {
			v = localName(i.Val)
		}
		return localName(i.Ok) + ", " + k + ", " + v + " = next " + localName(i.Iter)
	case *Print:
		return fmt.Sprintf("print(nl=%v) %s", i.Newline, vals(i.Args))
	case *Defer:
		return "defer " + InstrString(i.Call)
	case *Recover:
		return localName(i.Dst) + " = recover"
	case *Rebind:
		return "rebind " + localName(i.L)
	case *BoxParam:
		return "boxparam " + localName(i.L)
	case *Jump:
		return fmt.Sprintf("jump b%d", i.Target.ID)
	case *If:
		return fmt.Sprintf("if %s b%d else b%d", ValueString(i.Cond), i.Then.ID, i.Else.ID)
	case *Pause:
		return fmt.Sprintf("pause %s -> b%d", InstrString(i.Op), i.Next.ID)
	case *MakeChan:
		return localName(i.Dst) + " = makechan " + ValueString(i.Size)
	case *Send:
		return ValueString(i.Ch) + " <- " + ValueString(i.V)
	case *Recv:
		d := "_"
		if i.Dst != nil {
			d = localName(i.Dst)
		}
		return d + " = <-" + ValueString(i.Ch)
	case *Close:
		return "close " + ValueString(i.Ch)
	case *Select:
		return fmt.Sprintf("%s = select(%d cases, default=%v)", localName(i.Index), len(i.Cases), i.Default)
	case *Go:
		return "go " + InstrString(i.Call)
	case *Return:
		return "return"
	case *Panic:
		return "panic " + ValueString(i.X)
	case *PanicRuntime:
		return fmt.Sprintf("panic runtime %q", i.Msg)
	case *Unreachable:
		return "unreachable"
	}
	return fmt.Sprintf("?%T", in)
}

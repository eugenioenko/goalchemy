package lower

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/ir"
)

func (fl *fnLowerer) callExpr(e *ast.CallExpr) []ir.Value {
	fun := ast.Unparen(e.Fun)
	if tv := fl.info.Types[fun]; tv.IsType() {
		return []ir.Value{fl.conversion(e.Args[0], fl.typ(tv.Type), e)}
	}
	if id, ok := fun.(*ast.Ident); ok {
		if b, ok := fl.info.Uses[id].(*types.Builtin); ok {
			return fl.builtin(e, b.Name())
		}
	}
	if out, ok := fl.intrinsic(e); ok {
		return out
	}
	c := fl.prepareCall(e, false)
	sig := fl.info.TypeOf(fun).Underlying().(*types.Signature)
	var out []ir.Value
	for i := 0; i < sig.Results().Len(); i++ {
		d := fl.temp(fl.typ(sig.Results().At(i).Type()))
		c.Dsts = append(c.Dsts, d)
		out = append(out, d)
	}
	fl.emit(c)
	return out
}

// prepareCall evaluates the callee and arguments of a non-builtin call.
// In defer mode, interface calls become bound method values.
func (fl *fnLowerer) prepareCall(e *ast.CallExpr, deferMode bool) *ir.Call {
	fun := ast.Unparen(e.Fun)
	sig := fl.info.TypeOf(fun).Underlying().(*types.Signature)
	c := &ir.Call{At: at(e)}
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		if sel := fl.info.Selections[f]; sel != nil && sel.Kind() == types.MethodVal {
			m := sel.Obj().(*types.Func)
			if ext := fl.l.extern(m); ext != nil {
				recv, _ := fl.receiver(f.X, sel.Index()[:len(sel.Index())-1], m, f)
				args := fl.args(e, sig)
				c.Kind, c.Extern, c.Args = ir.CallExtern, ext, append([]ir.Value{recv}, args...)
				return c
			}
			recv, iface := fl.receiver(f.X, sel.Index()[:len(sel.Index())-1], m, f)
			args := fl.args(e, sig)
			if iface {
				if deferMode {
					bound := fl.temp(fl.typ(sig))
					fl.emit(&ir.MakeIfaceBound{At: at(f), Dst: bound, Recv: recv, Method: ir.MethodID(m)})
					c.Kind, c.Fn, c.Args = ir.CallValue, bound, args
					return c
				}
				c.Kind, c.Recv, c.Method, c.Args = ir.CallInterface, recv, ir.MethodID(m), args
				return c
			}
			c.Kind, c.Func, c.Args = ir.CallStatic, fl.l.declared(m), append([]ir.Value{recv}, args...)
			return c
		}
		if sel := fl.info.Selections[f]; sel == nil {
			if obj, ok := fl.info.Uses[f.Sel].(*types.Func); ok {
				return fl.staticCall(c, obj, e, sig)
			}
		}
	case *ast.Ident:
		if obj, ok := fl.info.Uses[f].(*types.Func); ok {
			return fl.staticCall(c, obj, e, sig)
		}
	}
	fv := fl.expr(fun)
	c.Kind, c.Fn, c.Args = ir.CallValue, fv, fl.args(e, sig)
	return c
}

func (fl *fnLowerer) staticCall(c *ir.Call, obj *types.Func, e *ast.CallExpr, sig *types.Signature) *ir.Call {
	if f, ok := fl.l.funcs[obj.Origin()]; ok {
		c.Kind, c.Func, c.Args = ir.CallStatic, f, fl.args(e, sig)
		return c
	}
	if ext := fl.l.extern(obj); ext != nil {
		c.Kind, c.Extern, c.Args = ir.CallExtern, ext, fl.args(e, sig)
		return c
	}
	fail(e.Pos(), "call of unavailable function %s", obj.FullName())
	return nil
}

func (l *Lowerer) extern(obj types.Object) *ir.Extern {
	if l.reg == nil {
		return nil
	}
	key := catalog.SymbolKey(obj)
	id, ok := l.reg.Symbols[key]
	if !ok {
		return nil
	}
	if ext, ok := l.out.Externals[id]; ok {
		return ext
	}
	var sig *ir.Type
	switch o := obj.(type) {
	case *types.Func:
		sig = l.funcSig(o.Signature())
	case *types.Var:
		sig = l.ts.Of(types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", o.Type())), false))
	}
	ext := &ir.Extern{Contract: id, Symbol: key, Sig: sig, MaySuspend: l.reg.Suspends[id]}
	l.out.Externals[id] = ext
	return ext
}

// args evaluates call arguments left to right, converting each to its
// parameter type and packing variadic arguments into a slice.
func (fl *fnLowerer) args(e *ast.CallExpr, sig *types.Signature) []ir.Value {
	params := sig.Params()
	np := params.Len()
	var vals []ir.Value
	var types_ []*ir.Type
	if len(e.Args) == 1 && np > 1 || len(e.Args) == 1 && isTupleCall(fl.info, e.Args[0]) {
		vals = fl.exprN(e.Args[0])
	} else {
		for i, a := range e.Args {
			pi := i
			if pi >= np {
				pi = np - 1
			}
			pt := fl.typ(params.At(pi).Type())
			if sig.Variadic() && pi == np-1 && !e.Ellipsis.IsValid() {
				pt = pt.U().Elem
			}
			vals = append(vals, fl.exprTo(a, pt))
		}
	}
	for i := range vals {
		pi := i
		if pi >= np {
			pi = np - 1
		}
		pt := fl.typ(params.At(pi).Type())
		if sig.Variadic() && pi == np-1 && !e.Ellipsis.IsValid() {
			pt = pt.U().Elem
		}
		types_ = append(types_, pt)
		vals[i] = fl.conv(vals[i], pt, e)
	}
	if !sig.Variadic() || e.Ellipsis.IsValid() {
		return vals
	}
	fixed := vals[:np-1]
	extra := vals[np-1:]
	st := fl.typ(params.At(np - 1).Type())
	var packed ir.Value
	if len(extra) == 0 {
		packed = &ir.Const{Type: st, Nil: true}
	} else {
		s := fl.temp(st)
		n := &ir.Const{Type: fl.ts().IntT(), Val: constant.MakeInt64(int64(len(extra)))}
		fl.emit(&ir.MakeSlice{At: at(e), Dst: s, Len: n, Cap: n})
		for i, v := range extra {
			ic := &ir.Const{Type: fl.ts().IntT(), Val: constant.MakeInt64(int64(i))}
			fl.emit(&ir.Store{At: at(e), Place: &ir.Place{Root: ir.SliceRoot{Slice: s, Index: ic}, Type: st.U().Elem}, V: v})
		}
		packed = s
	}
	return append(append([]ir.Value(nil), fixed...), packed)
}

func isTupleCall(info *types.Info, e ast.Expr) bool {
	_, ok := info.TypeOf(e).(*types.Tuple)
	return ok
}

// conversion lowers an explicit conversion T(x).
func (fl *fnLowerer) conversion(arg ast.Expr, t *ir.Type, e ast.Node) ir.Value {
	if tv, ok := fl.info.Types[arg]; ok && tv.IsNil() {
		return &ir.Const{Type: t, Nil: true}
	}
	v := fl.expr(arg)
	from := v.IRType()
	if from == t && t.U().Kind != ir.KFloat {
		return v
	}
	if t.IsInterface() {
		return fl.conv(v, t, e)
	}
	fu, tu := from.U(), t.U()
	kind := ir.ConvNop
	switch {
	case (fu.Kind == ir.KFloat && (tu.Kind == ir.KFloat || tu.Kind == ir.KInt)) ||
		(fu.Kind == ir.KInt && tu.Kind == ir.KFloat):
		kind = ir.ConvFloat
	case fu.Kind == ir.KInt && tu.Kind == ir.KInt:
		if fu.Int != tu.Int {
			kind = ir.ConvInt
		}
	case fu.Kind == ir.KInt && tu.Kind == ir.KString:
		kind = ir.ConvIntToString
	case fu.Kind == ir.KString && tu.Kind == ir.KSlice:
		if tu.Elem.U().Int == ir.U8 {
			kind = ir.ConvStringToBytes
		} else {
			kind = ir.ConvStringToRunes
		}
	case fu.Kind == ir.KSlice && tu.Kind == ir.KString:
		if fu.Elem.U().Int == ir.U8 {
			kind = ir.ConvBytesToString
		} else {
			kind = ir.ConvRunesToString
		}
	case fu.Kind == ir.KSlice && tu.Kind == ir.KArray:
		kind = ir.ConvSliceToArray
	}
	dst := fl.temp(t)
	fl.emit(&ir.Convert{At: at(e), Dst: dst, X: v, Kind: kind})
	return dst
}

func (fl *fnLowerer) builtin(e *ast.CallExpr, name string) []ir.Value {
	t := fl.info.TypeOf(e)
	switch name {
	case "len", "cap":
		v := fl.expr(e.Args[0])
		dst := fl.temp(fl.ts().IntT())
		if name == "len" {
			fl.emit(&ir.Len{At: at(e), Dst: dst, X: v})
		} else {
			fl.emit(&ir.Cap{At: at(e), Dst: dst, X: v})
		}
		return []ir.Value{dst}
	case "append":
		st := fl.typ(t)
		s := fl.exprTo(e.Args[0], st)
		ap := &ir.Append{At: at(e), Dst: fl.temp(st), S: s}
		if e.Ellipsis.IsValid() {
			sp := e.Args[1]
			if tv := fl.info.Types[sp]; tv.IsNil() {
				ap.Spread = &ir.Const{Type: st, Nil: true}
			} else {
				ap.Spread = fl.expr(sp)
			}
		} else {
			for _, a := range e.Args[1:] {
				ap.Elems = append(ap.Elems, fl.exprTo(a, st.U().Elem))
			}
		}
		fl.emit(ap)
		return []ir.Value{ap.Dst}
	case "copy":
		d := fl.expr(e.Args[0])
		s := fl.expr(e.Args[1])
		n := fl.temp(fl.ts().IntT())
		fl.emit(&ir.Copy{At: at(e), N: n, Dst: d, Src: s})
		return []ir.Value{n}
	case "delete":
		m := fl.expr(e.Args[0])
		k := fl.exprTo(e.Args[1], m.IRType().U().Key)
		fl.emit(&ir.MapDelete{At: at(e), M: m, K: k})
		return nil
	case "clear":
		fl.emit(&ir.Clear{At: at(e), X: fl.expr(e.Args[0])})
		return nil
	case "make":
		mt := fl.typ(t)
		switch mt.U().Kind {
		case ir.KSlice:
			n := fl.toInt(fl.expr(e.Args[1]), e)
			var c ir.Value = n
			if len(e.Args) > 2 {
				c = fl.toInt(fl.expr(e.Args[2]), e)
			}
			dst := fl.temp(mt)
			fl.emit(&ir.MakeSlice{At: at(e), Dst: dst, Len: n, Cap: c})
			return []ir.Value{dst}
		case ir.KMap:
			if len(e.Args) > 1 {
				fl.expr(e.Args[1])
			}
			dst := fl.temp(mt)
			fl.emit(&ir.MakeMap{At: at(e), Dst: dst})
			return []ir.Value{dst}
		case ir.KChan:
			return []ir.Value{fl.makeChan(e, mt)}
		}
	case "new":
		dst := fl.temp(fl.typ(t))
		fl.emit(&ir.New{At: at(e), Dst: dst})
		return []ir.Value{dst}
	case "panic":
		v := fl.exprTo(e.Args[0], fl.ts().Any())
		fl.term(&ir.Panic{At: at(e), X: v})
		return nil
	case "recover":
		dst := fl.temp(fl.ts().Any())
		fl.f.HasRecover = true
		fl.emit(&ir.Recover{At: at(e), Dst: dst})
		return []ir.Value{dst}
	case "print", "println":
		var vals []ir.Value
		if len(e.Args) == 1 && isTupleCall(fl.info, e.Args[0]) {
			vals = fl.exprN(e.Args[0])
		} else {
			for _, a := range e.Args {
				vals = append(vals, fl.expr(a))
			}
		}
		fl.emit(&ir.Print{At: at(e), Args: vals, Newline: name == "println"})
		return nil
	case "min", "max":
		rt := fl.typ(t)
		op := ir.Min
		if name == "max" {
			op = ir.Max
		}
		acc := fl.exprTo(e.Args[0], rt)
		for _, a := range e.Args[1:] {
			v := fl.exprTo(a, rt)
			dst := fl.temp(rt)
			fl.emit(&ir.BinOp{At: at(e), Dst: dst, Op: op, X: acc, Y: v})
			acc = dst
		}
		return []ir.Value{acc}
	case "close":
		fl.closeChan(e)
		return nil
	}
	fail(e.Pos(), "unsupported builtin %s", name)
	return nil
}

func (fl *fnLowerer) toInt(v ir.Value, n ast.Node) ir.Value {
	it := fl.ts().IntT()
	if v.IRType() == it {
		return v
	}
	if c, ok := v.(*ir.Const); ok {
		return &ir.Const{Type: it, Val: c.Val}
	}
	dst := fl.temp(it)
	kind := ir.ConvInt
	if v.IRType().U().Int == ir.I64 {
		kind = ir.ConvNop
	}
	fl.emit(&ir.Convert{At: at(n), Dst: dst, X: v, Kind: kind})
	return dst
}

func (fl *fnLowerer) deferStmt(s *ast.DeferStmt) {
	fl.f.HasDefer = true
	e := s.Call
	fun := ast.Unparen(e.Fun)
	if id, ok := fun.(*ast.Ident); ok {
		if b, ok := fl.info.Uses[id].(*types.Builtin); ok {
			fl.deferBuiltin(s, b.Name())
			return
		}
	}
	if tv := fl.info.Types[fun]; tv.IsType() {
		fail(s.Pos(), "defer of a conversion")
	}
	c := fl.prepareCall(e, true)
	fl.emit(&ir.Defer{At: at(s), Call: c})
}

// deferBuiltin evaluates the builtin's arguments now and defers a call to a
// synthesized function that performs the builtin.
func (fl *fnLowerer) deferBuiltin(s *ast.DeferStmt, name string) {
	if name == "recover" {
		return
	}
	fl.emit(&ir.Defer{At: at(s), Call: fl.builtinCall(s.Call, name, "defer")})
}

// builtinCall evaluates a builtin call's arguments and returns a call of a
// synthesized function performing the builtin, for defer and go.
func (fl *fnLowerer) builtinCall(call *ast.CallExpr, name, kind string) *ir.Call {
	s := call
	var vals []ir.Value
	for _, a := range call.Args {
		if name == "panic" {
			vals = append(vals, fl.exprTo(a, fl.ts().Any()))
		} else {
			vals = append(vals, fl.expr(a))
		}
	}
	var ptypes []*types.Var
	for _, v := range vals {
		ptypes = append(ptypes, types.NewVar(token.NoPos, nil, "", v.IRType().Go))
	}
	sig := fl.typ(types.NewSignatureType(nil, nil, nil, types.NewTuple(ptypes...), nil, false))
	f := fl.l.addFunc(&ir.Func{Name: fl.f.Name + "$" + kind + "_" + name, Sym: fl.l.sym(fl.f.Sym + "_" + kind + "_" + name), Pkg: fl.f.Pkg, Sig: sig, Wrapper: true})
	child := fl.l.newFn(fl.p, f)
	child.start()
	var args []ir.Value
	for _, v := range vals {
		p := f.NewLocal("", v.IRType(), ir.LParam)
		f.Params = append(f.Params, p)
		args = append(args, p)
	}
	switch name {
	case "print", "println":
		child.emit(&ir.Print{At: at(s), Args: args, Newline: name == "println"})
	case "panic":
		child.term(&ir.Panic{At: at(s), X: args[0]})
	case "delete":
		child.emit(&ir.MapDelete{At: at(s), M: args[0], K: child.conv(args[1], args[0].IRType().U().Key, s)})
	case "clear":
		child.emit(&ir.Clear{At: at(s), X: args[0]})
	case "copy":
		child.emit(&ir.Copy{At: at(s), N: child.temp(fl.ts().IntT()), Dst: args[0], Src: args[1]})
	case "close":
		child.closeValue(args[0], s)
	default:
		fail(s.Pos(), "%s of builtin %s", kind, name)
	}
	child.finish()
	return &ir.Call{At: at(s), Kind: ir.CallStatic, Func: f, Args: vals}
}

// methodExprFunc returns a function taking the receiver as its first
// parameter for the method expression sel on recvT.
func (l *Lowerer) methodExprFunc(recvT *ir.Type, sel *types.Selection) *ir.Func {
	return l.wrapper(recvT, sel, true)
}

// methodFunc returns the function implementing sel's method for receiver
// type t: the declared method when its receiver matches, else a wrapper.
func (l *Lowerer) methodFunc(t *ir.Type, sel *types.Selection) *ir.Func {
	return l.wrapper(t, sel, false)
}

// wrapper builds the receiver-adapting function. Method expressions report
// a nil pointer receiver of a value method with Go's panicwrap message;
// method-set wrappers dereference and fault like inlined dispatch.
func (l *Lowerer) wrapper(t *ir.Type, sel *types.Selection, methodExpr bool) *ir.Func {
	m := sel.Obj().(*types.Func)
	path := sel.Index()[:len(sel.Index())-1]
	if len(path) == 0 && !t.IsInterface() {
		recv := m.Signature().Recv()
		if types.Identical(recv.Type(), t.Go) {
			if ext := l.extern(m); ext != nil {
				return l.externWrapper(ext, m)
			}
			return l.declared(m)
		}
	}
	key := t.Go.String() + "|" + ir.MethodID(m) + "|" + itoa(len(path))
	if methodExpr {
		key += "|expr"
	}
	for _, i := range path {
		key += "." + itoa(i)
	}
	if f, ok := l.wrappers[key]; ok {
		return f
	}
	msig := m.Signature()
	var params []*types.Var
	params = append(params, types.NewVar(token.NoPos, nil, "recv", t.Go))
	for i := 0; i < msig.Params().Len(); i++ {
		params = append(params, msig.Params().At(i))
	}
	sig := l.ts.Of(types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), msig.Results(), msig.Variadic()))
	name := t.String() + "." + m.Name()
	f := l.addFunc(&ir.Func{Name: name, Sym: l.sym("wrap_" + sanitize(t.String()) + "_" + m.Name()), Sig: sig,
		Pkg: l.receiverPackage(t, m), Wrapper: true, MethodID: ir.MethodID(m), RecvType: t})
	l.wrappers[key] = f
	fl := l.newFn(nil, f)
	fl.start()
	recv := f.NewLocal("recv", t, ir.LParam)
	f.Params = append(f.Params, recv)
	f.Recv = recv
	var args []ir.Value
	for i := 0; i < msig.Params().Len(); i++ {
		p := f.NewLocal("", l.ts.Of(msig.Params().At(i).Type()), ir.LParam)
		f.Params = append(f.Params, p)
		args = append(args, p)
	}
	for i := 0; i < msig.Results().Len(); i++ {
		f.Results = append(f.Results, f.NewLocal("", l.ts.Of(msig.Results().At(i).Type()), ir.LResult))
	}
	for _, r := range f.Results {
		fl.emit(&ir.DeclVar{L: r})
	}
	wantPtr := false
	if r := msig.Recv(); r != nil {
		_, wantPtr = r.Type().(*types.Pointer)
		if types.IsInterface(r.Type()) {
			wantPtr = false
		}
	}
	var rv ir.Value
	iface := false
	switch {
	case t.IsInterface():
		rv, iface = recv, true
	case t.U().Kind == ir.KPointer:
		if len(path) == 0 && !wantPtr && methodExpr {
			// A value method called through a pointer: Go panics with a
			// specific message for nil.
			isNil := fl.temp(l.ts.Bool())
			fl.emit(&ir.BinOp{Dst: isNil, Op: ir.Eq, X: recv, Y: &ir.Const{Type: t, Nil: true}})
			bad := f.NewBlock("nilrecv")
			ok := f.NewBlock("ok")
			fl.term(&ir.If{Cond: isNil, Then: bad, Else: ok})
			fl.b = bad
			elem := t.U().Elem
			fl.term(&ir.PanicRuntime{Msg: "value method " + elem.String() + "." + m.Name() + " called using nil *" + elem.Obj + " pointer"})
			fl.b = ok
		}
		if len(path) == 0 && wantPtr {
			rv = recv
		} else {
			rv, iface = fl.adaptRecv(&ir.Place{Root: ir.DerefRoot{Ptr: recv}, Type: t.U().Elem}, path, wantPtr, nil)
		}
	default:
		rv, iface = fl.adaptRecv(localPlace(recv), path, wantPtr, nil)
	}
	c := &ir.Call{Dsts: append([]*ir.Local(nil), f.Results...), Args: args}
	if iface {
		c.Kind, c.Recv, c.Method = ir.CallInterface, rv, ir.MethodID(m)
	} else {
		if ext := l.extern(m); ext != nil {
			c.Kind, c.Extern = ir.CallExtern, ext
		} else {
			c.Kind, c.Func = ir.CallStatic, l.declared(m)
		}
		c.Args = append([]ir.Value{rv}, args...)
	}
	// Results are written through temps so stores keep value semantics.
	var outs []*ir.Local
	for _, r := range f.Results {
		outs = append(outs, fl.temp(r.Type))
	}
	c.Dsts = outs
	fl.emit(c)
	for i, r := range f.Results {
		fl.emit(&ir.Store{Place: localPlace(r), V: outs[i]})
	}
	fl.term(&ir.Return{})
	fl.finish()
	return f
}

// receiverPackage keeps receiver adapters with their source type, including
// promoted methods whose implementation belongs to an imported package.
func (l *Lowerer) receiverPackage(t *ir.Type, method *types.Func) string {
	if t.Kind == ir.KPointer {
		t = t.Elem
	}
	if t.Kind == ir.KNamed && t.Pkg != "" {
		for _, p := range l.prog.Source {
			if p.PkgPath == t.Pkg {
				return t.Pkg
			}
		}
	}
	if p := l.pkgs[method.Pkg()]; p != nil {
		return p.PkgPath
	}
	return ""
}

// externWrapper creates a function value wrapper for an external capability.
func (l *Lowerer) externWrapper(ext *ir.Extern, obj *types.Func) *ir.Func {
	key := "extern|" + ext.Contract
	if f, ok := l.wrappers[key]; ok {
		return f
	}
	sig := obj.Signature()
	f := l.addFunc(&ir.Func{Name: obj.FullName(), Sym: l.sym("extern_" + sanitize(ext.Contract)), Sig: ext.Sig, Wrapper: true})
	l.wrappers[key] = f
	fl := l.newFn(nil, f)
	fl.start()
	var args []ir.Value
	if recv := sig.Recv(); recv != nil {
		p := f.NewLocal("recv", l.ts.Of(recv.Type()), ir.LParam)
		f.Params = append(f.Params, p)
		f.Recv = p
		args = append(args, p)
	}
	for i := 0; i < sig.Params().Len(); i++ {
		p := f.NewLocal("", l.ts.Of(sig.Params().At(i).Type()), ir.LParam)
		f.Params = append(f.Params, p)
		args = append(args, p)
	}
	var outs []*ir.Local
	for i := 0; i < sig.Results().Len(); i++ {
		r := f.NewLocal("", l.ts.Of(sig.Results().At(i).Type()), ir.LResult)
		f.Results = append(f.Results, r)
		fl.emit(&ir.DeclVar{L: r})
		outs = append(outs, fl.temp(r.Type))
	}
	fl.emit(&ir.Call{Kind: ir.CallExtern, Extern: ext, Args: args, Dsts: outs})
	for i, r := range f.Results {
		fl.emit(&ir.Store{Place: localPlace(r), V: outs[i]})
	}
	fl.term(&ir.Return{})
	fl.finish()
	return f
}

func (fl *fnLowerer) makeChan(e *ast.CallExpr, t *ir.Type) ir.Value {
	var size ir.Value = fl.intConst(0)
	if len(e.Args) > 1 {
		size = fl.toInt(fl.expr(e.Args[1]), e)
	}
	dst := fl.temp(t)
	fl.emit(&ir.MakeChan{At: at(e), Dst: dst, Size: size})
	return dst
}

func (fl *fnLowerer) closeChan(e *ast.CallExpr) {
	fl.closeValue(fl.expr(e.Args[0]), e)
}

func (fl *fnLowerer) closeValue(v ir.Value, n ast.Node) {
	fl.emit(&ir.Close{At: at(n), Ch: v})
}

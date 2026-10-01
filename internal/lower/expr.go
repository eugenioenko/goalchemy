package lower

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"goalchemy/internal/ir"
)

// expr lowers an expression producing exactly one value.
func (fl *fnLowerer) expr(e ast.Expr) ir.Value {
	vals := fl.exprN1(e, 1)
	if len(vals) != 1 {
		fail(e.Pos(), "expression produces %d values where one is needed", len(vals))
	}
	return vals[0]
}

// exprTo lowers e and converts it to type t, typing untyped nil from t.
func (fl *fnLowerer) exprTo(e ast.Expr, t *ir.Type) ir.Value {
	if tv, ok := fl.info.Types[e]; ok && tv.IsNil() {
		return &ir.Const{Type: t, Nil: true}
	}
	return fl.conv(fl.expr(e), t, e)
}

// exprN lowers an expression statement or tuple-producing expression.
func (fl *fnLowerer) exprN(e ast.Expr) []ir.Value { return fl.exprN1(e, -1) }

// exprN1 lowers e; want is the number of values the context needs (2
// selects comma-ok forms), or -1 for any.
func (fl *fnLowerer) exprN1(e ast.Expr, want int) []ir.Value {
	tv := fl.info.Types[e]
	if tv.Value != nil && !tv.IsType() {
		return []ir.Value{fl.constant(tv.Type, tv.Value)}
	}
	switch x := e.(type) {
	case *ast.ParenExpr:
		return fl.exprN1(x.X, want)
	case *ast.Ident:
		return []ir.Value{fl.ident(x)}
	case *ast.BasicLit:
		return []ir.Value{fl.constant(tv.Type, tv.Value)}
	case *ast.BinaryExpr:
		return []ir.Value{fl.binary(x)}
	case *ast.UnaryExpr:
		return fl.unary(x, want)
	case *ast.StarExpr:
		return []ir.Value{fl.load(&ir.Place{Root: ir.DerefRoot{Ptr: fl.expr(x.X)}, Type: fl.typeOf(x)}, x)}
	case *ast.SelectorExpr:
		return []ir.Value{fl.selector(x)}
	case *ast.IndexExpr:
		return fl.index(x, want)
	case *ast.SliceExpr:
		return []ir.Value{fl.sliceExpr(x)}
	case *ast.TypeAssertExpr:
		t := fl.typeOf(x)
		v := fl.expr(x.X)
		fl.markBoxed(t)
		dst := fl.temp(t)
		ta := &ir.TypeAssert{At: at(x), Dst: dst, X: v, T: t}
		if want == 2 {
			ta.Ok = fl.temp(fl.ts().Bool())
			fl.emit(ta)
			return []ir.Value{dst, ta.Ok}
		}
		fl.emit(ta)
		return []ir.Value{dst}
	case *ast.CallExpr:
		return fl.callExpr(x)
	case *ast.CompositeLit:
		return []ir.Value{fl.compositeLit(x)}
	case *ast.FuncLit:
		return []ir.Value{fl.funcLit(x)}
	}
	fail(e.Pos(), "unsupported expression %T", e)
	return nil
}

func (fl *fnLowerer) constant(t types.Type, v constant.Value) ir.Value {
	if b, ok := t.(*types.Basic); ok && b.Info()&types.IsUntyped != 0 {
		t = types.Default(t)
	}
	return &ir.Const{Type: fl.typ(t), Val: v}
}

func (fl *fnLowerer) markBoxed(t *ir.Type) {
	if !t.IsInterface() {
		t.Boxed = true
	}
}

// snap copies a value read from a live variable into a temp so later
// stores cannot change it.
func (fl *fnLowerer) snap(v ir.Value) ir.Value {
	if l, ok := v.(*ir.Local); ok && l.Kind != ir.LTemp {
		t := fl.temp(l.Type)
		fl.emit(&ir.Assign{Dst: t, Src: l})
		return t
	}
	return v
}

func (fl *fnLowerer) load(p *ir.Place, n ast.Node) *ir.Local {
	dst := fl.temp(p.Type)
	fl.emit(&ir.Load{At: at(n), Dst: dst, Place: p})
	return dst
}

func (fl *fnLowerer) ident(id *ast.Ident) ir.Value {
	obj := fl.info.Uses[id]
	if obj == nil {
		obj = fl.info.Defs[id]
	}
	switch o := obj.(type) {
	case *types.Var:
		if g, ok := fl.l.globals[o]; ok {
			return fl.load(&ir.Place{Root: ir.GlobalRoot{Global: g}, Type: g.Type}, id)
		}
		loc, ok := fl.locals[o]
		if !ok {
			fail(id.Pos(), "variable %s used before declaration", o.Name())
		}
		if !loc.Boxed && !loc.Type.IsAggregate() {
			return loc
		}
		return fl.load(localPlace(loc), id)
	case *types.Func:
		return fl.funcValue(o, id)
	case *types.Nil:
		t := fl.info.TypeOf(id)
		if b, ok := t.(*types.Basic); ok && b.Kind() == types.UntypedNil {
			fail(id.Pos(), "untyped nil without a context type")
		}
		return &ir.Const{Type: fl.typ(t), Nil: true}
	}
	fail(id.Pos(), "unsupported identifier %s", id.Name)
	return nil
}

func (fl *fnLowerer) funcValue(o *types.Func, n ast.Node) ir.Value {
	if f, ok := fl.l.funcs[o.Origin()]; ok {
		return &ir.FuncRef{Func: f, Type: f.Sig}
	}
	if ext := fl.l.extern(o); ext != nil {
		return &ir.FuncRef{Func: fl.l.externWrapper(ext, o), Type: fl.typ(o.Type())}
	}
	fail(n.Pos(), "function %s is not available", o.FullName())
	return nil
}

// conv applies Go's implicit conversion of v to type t.
func (fl *fnLowerer) conv(v ir.Value, t *ir.Type, n ast.Node) ir.Value {
	vt := v.IRType()
	if vt == t {
		return v
	}
	if c, ok := v.(*ir.Const); ok && c.Nil {
		return &ir.Const{Type: t, Nil: true}
	}
	if t.IsInterface() {
		dst := fl.temp(t)
		if vt.IsInterface() {
			fl.emit(&ir.Convert{At: at(n), Dst: dst, X: v, Kind: ir.ConvIfaceToIface})
		} else {
			fl.markBoxed(vt)
			fl.emit(&ir.MakeInterface{At: at(n), Dst: dst, X: v})
		}
		return dst
	}
	if c, ok := v.(*ir.Const); ok {
		return &ir.Const{Type: t, Val: c.Val}
	}
	dst := fl.temp(t)
	fl.emit(&ir.Convert{At: at(n), Dst: dst, X: v, Kind: ir.ConvNop})
	return dst
}

var binOps = map[token.Token]ir.BinOpKind{
	token.ADD: ir.Add, token.SUB: ir.Sub, token.MUL: ir.Mul, token.QUO: ir.Div, token.REM: ir.Rem,
	token.AND: ir.And, token.OR: ir.Or, token.XOR: ir.Xor, token.AND_NOT: ir.AndNot,
	token.SHL: ir.Shl, token.SHR: ir.Shr,
	token.EQL: ir.Eq, token.NEQ: ir.Ne, token.LSS: ir.Lt, token.LEQ: ir.Le, token.GTR: ir.Gt, token.GEQ: ir.Ge,
}

var assignOps = map[token.Token]token.Token{
	token.ADD_ASSIGN: token.ADD, token.SUB_ASSIGN: token.SUB, token.MUL_ASSIGN: token.MUL,
	token.QUO_ASSIGN: token.QUO, token.REM_ASSIGN: token.REM, token.AND_ASSIGN: token.AND,
	token.OR_ASSIGN: token.OR, token.XOR_ASSIGN: token.XOR, token.AND_NOT_ASSIGN: token.AND_NOT,
	token.SHL_ASSIGN: token.SHL, token.SHR_ASSIGN: token.SHR,
}

func (fl *fnLowerer) binary(x *ast.BinaryExpr) ir.Value {
	switch x.Op {
	case token.LAND, token.LOR:
		res := fl.temp(fl.typeOf(x))
		t := fl.f.NewBlock("bool.true")
		f := fl.f.NewBlock("bool.false")
		done := fl.f.NewBlock("bool.done")
		fl.cond(x, t, f)
		fl.b = t
		fl.emit(&ir.Assign{Dst: res, Src: &ir.Const{Type: res.Type, Val: constant.MakeBool(true)}})
		fl.jump(done)
		fl.b = f
		fl.emit(&ir.Assign{Dst: res, Src: &ir.Const{Type: res.Type, Val: constant.MakeBool(false)}})
		fl.jump(done)
		fl.b = done
		return res
	case token.EQL, token.NEQ:
		xt, yt := fl.operandType(x.X, x.Y), fl.operandType(x.Y, x.X)
		xv := fl.exprTo(x.X, xt)
		yv := fl.exprTo(x.Y, yt)
		eq := fl.equal(xv, xt, yv, yt, x)
		if x.Op == token.NEQ {
			ne := fl.temp(fl.ts().Bool())
			fl.emit(&ir.UnOp{At: at(x), Dst: ne, Op: ir.Not, X: eq})
			return ne
		}
		return eq
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		xt := fl.operandType(x.X, x.Y)
		xv := fl.exprTo(x.X, xt)
		yv := fl.exprTo(x.Y, xt)
		dst := fl.temp(fl.ts().Bool())
		fl.emit(&ir.BinOp{At: at(x), Dst: dst, Op: binOps[x.Op], X: xv, Y: yv})
		return dst
	case token.SHL, token.SHR:
		t := fl.typeOf(x)
		xv := fl.exprTo(x.X, t)
		yv := fl.expr(x.Y)
		dst := fl.temp(t)
		fl.emit(&ir.BinOp{At: at(x), Dst: dst, Op: binOps[x.Op], X: xv, Y: yv})
		return dst
	}
	op, ok := binOps[x.Op]
	if !ok {
		fail(x.OpPos, "unsupported operator %s", x.Op)
	}
	t := fl.typeOf(x)
	xv := fl.exprTo(x.X, t)
	yv := fl.exprTo(x.Y, t)
	dst := fl.temp(t)
	fl.emit(&ir.BinOp{At: at(x), Dst: dst, Op: op, X: xv, Y: yv})
	return dst
}

// operandType picks the comparison type for e, using other when e is an
// untyped nil or constant.
func (fl *fnLowerer) operandType(e, other ast.Expr) *ir.Type {
	t := fl.info.TypeOf(e)
	if b, ok := t.(*types.Basic); ok && b.Info()&types.IsUntyped != 0 {
		ot := fl.info.TypeOf(other)
		if ob, ok := ot.(*types.Basic); ok && ob.Info()&types.IsUntyped != 0 {
			return fl.typ(types.Default(t))
		}
		return fl.typ(ot)
	}
	return fl.typ(t)
}

// equal compares values for equality, boxing a concrete operand compared
// with an interface.
func (fl *fnLowerer) equal(x ir.Value, xt *ir.Type, y ir.Value, yt *ir.Type, n ast.Node) ir.Value {
	if xt.IsInterface() && !yt.IsInterface() {
		y = fl.conv(y, xt, n)
	} else if yt.IsInterface() && !xt.IsInterface() {
		x = fl.conv(x, yt, n)
	} else if xt != yt {
		if c, ok := y.(*ir.Const); ok {
			y = fl.conv(c, xt, n)
		} else if c, ok := x.(*ir.Const); ok {
			x = fl.conv(c, yt, n)
		}
	}
	dst := fl.temp(fl.ts().Bool())
	fl.emit(&ir.BinOp{At: at(n), Dst: dst, Op: ir.Eq, X: x, Y: y})
	return dst
}

func (fl *fnLowerer) unary(x *ast.UnaryExpr, want int) []ir.Value {
	switch x.Op {
	case token.ADD:
		return []ir.Value{fl.exprTo(x.X, fl.typeOf(x))}
	case token.SUB, token.XOR, token.NOT:
		t := fl.typeOf(x)
		v := fl.exprTo(x.X, t)
		dst := fl.temp(t)
		op := map[token.Token]ir.UnOpKind{token.SUB: ir.Neg, token.XOR: ir.BitNot, token.NOT: ir.Not}[x.Op]
		fl.emit(&ir.UnOp{At: at(x), Dst: dst, Op: op, X: v})
		return []ir.Value{dst}
	case token.AND:
		return []ir.Value{fl.addrOf(x.X, x)}
	case token.ARROW:
		return fl.recvExpr(x, want)
	}
	fail(x.OpPos, "unsupported unary operator %s", x.Op)
	return nil
}

func (fl *fnLowerer) addrOf(e ast.Expr, n ast.Node) ir.Value {
	e = ast.Unparen(e)
	if lit, ok := e.(*ast.CompositeLit); ok {
		pt := fl.ts().PointerTo(fl.typeOf(lit))
		p := fl.temp(pt)
		fl.emit(&ir.New{At: at(n), Dst: p})
		fl.fillComposite(lit, &ir.Place{Root: ir.DerefRoot{Ptr: p}, Type: pt.Elem})
		return p
	}
	pl := fl.place(e, true)
	if lr, ok := pl.Root.(ir.LocalRoot); ok && len(pl.Path) == 0 && !lr.Local.Boxed {
		fail(n.Pos(), "address of unboxed local %s", lr.Local.Name)
	}
	if pl.HasElement() {
		fail(n.Pos(), "address of a slice or array element")
	}
	dst := fl.temp(fl.ts().PointerTo(pl.Type))
	fl.emit(&ir.AddrOf{At: at(n), Dst: dst, Place: pl})
	return dst
}

// place lowers an addressable expression (or, when addr is false, any
// expression) to a storage designation, evaluating index and pointer
// operands left to right.
func (fl *fnLowerer) place(e ast.Expr, addr bool) *ir.Place {
	e = ast.Unparen(e)
	t := fl.typeOf(e)
	switch x := e.(type) {
	case *ast.Ident:
		switch o := fl.info.Uses[x].(type) {
		case *types.Var:
			if g, ok := fl.l.globals[o]; ok {
				return &ir.Place{Root: ir.GlobalRoot{Global: g}, Type: g.Type}
			}
			if loc, ok := fl.locals[o]; ok {
				return localPlace(loc)
			}
		}
	case *ast.SelectorExpr:
		sel := fl.info.Selections[x]
		if sel == nil {
			if v, ok := fl.info.Uses[x.Sel].(*types.Var); ok {
				if g, ok := fl.l.globals[v]; ok {
					return &ir.Place{Root: ir.GlobalRoot{Global: g}, Type: g.Type}
				}
			}
			break
		}
		if sel.Kind() != types.FieldVal {
			break
		}
		base := fl.basePlace(x.X, addr)
		return fl.fieldPath(base, sel.Index(), x)
	case *ast.IndexExpr:
		xt := fl.info.TypeOf(x.X).Underlying()
		switch u := xt.(type) {
		case *types.Slice:
			s := fl.expr(x.X)
			i := fl.expr(x.Index)
			return &ir.Place{Root: ir.SliceRoot{Slice: s, Index: i}, Type: t}
		case *types.Array:
			base := fl.place(x.X, addr)
			i := fl.expr(x.Index)
			return withProj(base, ir.Proj{Index: i, Type: t})
		case *types.Pointer:
			p := fl.expr(x.X)
			i := fl.expr(x.Index)
			base := &ir.Place{Root: ir.DerefRoot{Ptr: p}, Type: fl.typ(u.Elem())}
			return withProj(base, ir.Proj{Index: i, Type: t})
		}
	case *ast.StarExpr:
		return &ir.Place{Root: ir.DerefRoot{Ptr: fl.expr(x.X)}, Type: t}
	}
	if addr {
		fail(e.Pos(), "expression is not addressable")
	}
	return &ir.Place{Root: ir.ValueRoot{Value: fl.expr(e)}, Type: t}
}

// basePlace returns the storage of the struct designated by x for a field
// selection: x itself, or *x when x is a pointer.
func (fl *fnLowerer) basePlace(x ast.Expr, addr bool) *ir.Place {
	xt := fl.typeOf(x)
	if xt.U().Kind == ir.KPointer {
		return &ir.Place{Root: ir.DerefRoot{Ptr: fl.expr(x)}, Type: xt.U().Elem}
	}
	tv := fl.info.Types[x]
	return fl.place(x, addr && tv.Addressable())
}

func withProj(p *ir.Place, pr ir.Proj) *ir.Place {
	np := &ir.Place{Root: p.Root, Path: append(append([]ir.Proj(nil), p.Path...), pr), Type: pr.Type}
	return np
}

// fieldPath applies a selection path of field indices, dereferencing
// embedded pointers as needed.
func (fl *fnLowerer) fieldPath(p *ir.Place, path []int, n ast.Node) *ir.Place {
	for _, idx := range path {
		p = fl.derefIfPointer(p, n)
		st := p.Type.U()
		if st.Kind != ir.KStruct {
			fail(n.Pos(), "field selection on non-struct %s", p.Type)
		}
		p = withProj(p, ir.Proj{Field: idx, Type: st.Fields[idx].Type})
	}
	return p
}

func (fl *fnLowerer) derefIfPointer(p *ir.Place, n ast.Node) *ir.Place {
	if p.Type.U().Kind == ir.KPointer {
		v := fl.load(p, n)
		return &ir.Place{Root: ir.DerefRoot{Ptr: v}, Type: p.Type.U().Elem}
	}
	return p
}

func (fl *fnLowerer) selector(x *ast.SelectorExpr) ir.Value {
	sel := fl.info.Selections[x]
	if sel == nil {
		switch o := fl.info.Uses[x.Sel].(type) {
		case *types.Var:
			return fl.ident(x.Sel)
		case *types.Func:
			return fl.funcValue(o, x)
		}
		fail(x.Pos(), "unsupported qualified identifier %s", x.Sel.Name)
	}
	switch sel.Kind() {
	case types.FieldVal:
		return fl.load(fl.place(x, false), x)
	case types.MethodVal:
		m := sel.Obj().(*types.Func)
		recv, iface := fl.receiver(x.X, sel.Index()[:len(sel.Index())-1], m, x)
		t := fl.typeOf(x)
		dst := fl.temp(t)
		if iface {
			fl.emit(&ir.MakeIfaceBound{At: at(x), Dst: dst, Recv: recv, Method: m.Id()})
		} else {
			fl.emit(&ir.MakeBound{At: at(x), Dst: dst, Func: fl.l.declared(m), Recv: recv})
		}
		return dst
	case types.MethodExpr:
		recvT := fl.typ(sel.Recv())
		f := fl.l.methodExprFunc(recvT, sel)
		return &ir.FuncRef{Func: f, Type: fl.typeOf(x)}
	}
	fail(x.Pos(), "unsupported selector")
	return nil
}

// receiver evaluates the receiver for calling m on x through the implicit
// embedded field path. It reports iface when the method is reached through
// an interface value, in which case the returned value is that interface.
func (fl *fnLowerer) receiver(x ast.Expr, path []int, m *types.Func, n ast.Node) (ir.Value, bool) {
	xt := fl.typeOf(x)
	wantPtr := false
	if r := m.Signature().Recv(); r != nil {
		_, wantPtr = r.Type().(*types.Pointer)
		if types.IsInterface(r.Type()) {
			wantPtr = false
		}
	}
	if xt.IsInterface() && len(path) == 0 {
		return fl.expr(x), true
	}
	var p *ir.Place
	if xt.U().Kind == ir.KPointer {
		pv := fl.expr(x)
		if len(path) == 0 && wantPtr {
			return pv, false
		}
		p = &ir.Place{Root: ir.DerefRoot{Ptr: pv}, Type: xt.U().Elem}
	} else {
		p = fl.place(x, fl.info.Types[x].Addressable())
	}
	return fl.adaptRecv(p, path, wantPtr, n)
}

// adaptRecv walks path from p and produces the receiver value.
func (fl *fnLowerer) adaptRecv(p *ir.Place, path []int, wantPtr bool, n ast.Node) (ir.Value, bool) {
	p = fl.fieldPath(p, path, n)
	if p.Type.IsInterface() {
		return fl.load(p, n), true
	}
	if p.Type.U().Kind == ir.KPointer && p.Type.Kind != ir.KNamed {
		v := fl.load(p, n)
		if wantPtr {
			return v, false
		}
		return fl.load(&ir.Place{Root: ir.DerefRoot{Ptr: v}, Type: p.Type.U().Elem}, n), false
	}
	if wantPtr {
		if dr, ok := p.Root.(ir.DerefRoot); ok && len(p.Path) == 0 {
			return dr.Ptr, false
		}
		dst := fl.temp(fl.ts().PointerTo(p.Type))
		fl.emit(&ir.AddrOf{At: at(n), Dst: dst, Place: p})
		return dst, false
	}
	return fl.load(p, n), false
}

func (fl *fnLowerer) index(x *ast.IndexExpr, want int) []ir.Value {
	xt := fl.info.TypeOf(x.X).Underlying()
	switch u := xt.(type) {
	case *types.Map:
		m := fl.expr(x.X)
		k := fl.exprTo(x.Index, fl.typ(u.Key()))
		dst := fl.temp(fl.typ(u.Elem()))
		ml := &ir.MapLookup{At: at(x), Dst: dst, M: m, K: k}
		if want == 2 {
			ml.Ok = fl.temp(fl.ts().Bool())
			fl.emit(ml)
			return []ir.Value{dst, ml.Ok}
		}
		fl.emit(ml)
		return []ir.Value{dst}
	case *types.Basic:
		s := fl.expr(x.X)
		i := fl.expr(x.Index)
		dst := fl.temp(fl.ts().Basic(types.Uint8))
		fl.emit(&ir.IndexString{At: at(x), Dst: dst, S: s, I: i})
		return []ir.Value{dst}
	}
	return []ir.Value{fl.load(fl.place(x, false), x)}
}

func (fl *fnLowerer) sliceExpr(x *ast.SliceExpr) ir.Value {
	t := fl.typeOf(x)
	xt := fl.typeOf(x.X)
	var base ir.Value
	if xt.U().Kind == ir.KArray {
		pl := fl.place(x.X, true)
		p := fl.temp(fl.ts().PointerTo(pl.Type))
		fl.emit(&ir.AddrOf{At: at(x), Dst: p, Place: pl})
		base = p
	} else {
		base = fl.expr(x.X)
	}
	op := &ir.SliceOp{At: at(x), Dst: fl.temp(t), X: base}
	if x.Low != nil {
		op.Lo = fl.expr(x.Low)
	}
	if x.High != nil {
		op.Hi = fl.expr(x.High)
	}
	if x.Max != nil {
		op.Max = fl.expr(x.Max)
	}
	fl.emit(op)
	return op.Dst
}

func (fl *fnLowerer) compositeLit(lit *ast.CompositeLit) ir.Value {
	t := fl.typeOf(lit)
	if t.U().Kind == ir.KPointer {
		p := fl.temp(t)
		fl.emit(&ir.New{At: at(lit), Dst: p})
		fl.fillComposite(lit, &ir.Place{Root: ir.DerefRoot{Ptr: p}, Type: t.U().Elem})
		return p
	}
	switch t.U().Kind {
	case ir.KSlice:
		n := int64(0)
		idx := int64(0)
		for _, e := range lit.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				idx = constInt(fl.info.Types[kv.Key].Value)
			}
			idx++
			if idx > n {
				n = idx
			}
		}
		s := fl.temp(t)
		nc := &ir.Const{Type: fl.ts().IntT(), Val: constant.MakeInt64(n)}
		fl.emit(&ir.MakeSlice{At: at(lit), Dst: s, Len: nc, Cap: nc})
		idx = 0
		for _, e := range lit.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				idx = constInt(fl.info.Types[kv.Key].Value)
				e = kv.Value
			}
			v := fl.exprTo(e, t.U().Elem)
			ic := &ir.Const{Type: fl.ts().IntT(), Val: constant.MakeInt64(idx)}
			fl.emit(&ir.Store{At: at(e), Place: &ir.Place{Root: ir.SliceRoot{Slice: s, Index: ic}, Type: t.U().Elem}, V: v})
			idx++
		}
		return s
	case ir.KMap:
		m := fl.temp(t)
		fl.emit(&ir.MakeMap{At: at(lit), Dst: m})
		for _, e := range lit.Elts {
			kv := e.(*ast.KeyValueExpr)
			k := fl.exprTo(kv.Key, t.U().Key)
			v := fl.exprTo(kv.Value, t.U().Elem)
			fl.emit(&ir.MapStore{At: at(e), M: m, K: k, V: v})
		}
		return m
	}
	tmp := fl.temp(t)
	fl.emit(&ir.Zero{At: at(lit), Dst: tmp})
	fl.fillComposite(lit, localPlace(tmp))
	return tmp
}

func constInt(v constant.Value) int64 {
	i, _ := constant.Int64Val(constant.ToInt(v))
	return i
}

// fillComposite stores struct or array literal elements into p, which
// already holds a zero value.
func (fl *fnLowerer) fillComposite(lit *ast.CompositeLit, p *ir.Place) {
	t := p.Type.U()
	switch t.Kind {
	case ir.KStruct:
		st := fl.info.TypeOf(lit)
		if ptr, ok := st.Underlying().(*types.Pointer); ok {
			st = ptr.Elem()
		}
		gst := st.Underlying().(*types.Struct)
		for i, e := range lit.Elts {
			idx := i
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				name := kv.Key.(*ast.Ident).Name
				for j := 0; j < gst.NumFields(); j++ {
					if gst.Field(j).Name() == name {
						idx = j
					}
				}
				e = kv.Value
			}
			ft := t.Fields[idx].Type
			v := fl.exprTo(e, ft)
			fl.emit(&ir.Store{At: at(e), Place: withProj(p, ir.Proj{Field: idx, Type: ft}), V: v})
		}
	case ir.KArray:
		idx := int64(0)
		for _, e := range lit.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				idx = constInt(fl.info.Types[kv.Key].Value)
				e = kv.Value
			}
			v := fl.exprTo(e, t.Elem)
			ic := &ir.Const{Type: fl.ts().IntT(), Val: constant.MakeInt64(idx)}
			fl.emit(&ir.Store{At: at(e), Place: withProj(p, ir.Proj{Index: ic, Type: t.Elem}), V: v})
			idx++
		}
	default:
		if len(lit.Elts) > 0 {
			fail(lit.Pos(), "unsupported composite literal of %s", p.Type)
		}
	}
}

func (fl *fnLowerer) funcLit(lit *ast.FuncLit) ir.Value {
	fl.litN++
	sig := fl.info.TypeOf(lit).(*types.Signature)
	f := fl.l.addFunc(&ir.Func{Closure: true, Pkg: fl.f.Pkg, Pos: lit.Pos(),
		Name: fl.f.Name + "$" + itoa(fl.litN), Sym: fl.l.sym(fl.f.Sym + "_func" + itoa(fl.litN)),
		Sig: fl.typ(sig)})
	child := fl.l.newFn(fl.p, f)
	var env []*ir.Local
	for _, v := range fl.l.captures[lit] {
		outer, ok := fl.locals[v]
		if !ok {
			fail(lit.Pos(), "captured variable %s is not in scope", v.Name())
		}
		inner := f.NewLocal(v.Name(), outer.Type, ir.LEnv)
		inner.Boxed = true
		child.locals[v] = inner
		f.Env = append(f.Env, inner)
		env = append(env, outer)
	}
	child.params(lit.Type.Params, tupleVars(sig.Params()), ir.LParam)
	child.results(lit.Type.Results, sig.Results())
	child.body(lit.Body)
	dst := fl.temp(f.Sig)
	fl.emit(&ir.MakeClosure{At: at(lit), Dst: dst, Func: f, Env: env})
	return dst
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func (fl *fnLowerer) recvExpr(x *ast.UnaryExpr, want int) []ir.Value {
	fail(x.Pos(), "channel receive requires the cooperative gate")
	return nil
}

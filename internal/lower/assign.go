package lower

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"goalchemy/internal/ir"
)

type lhsKind uint8

const (
	lhsBlank lhsKind = iota
	lhsNew
	lhsPlace
	lhsMap
)

type lhsTarget struct {
	kind  lhsKind
	local *ir.Local
	place *ir.Place
	m, k  ir.Value
	t     *ir.Type
	node  ast.Node
}

// lhs evaluates the operands of an assignment destination.
func (fl *fnLowerer) lhs(e ast.Expr, define bool, snap bool) lhsTarget {
	e = ast.Unparen(e)
	if id, ok := e.(*ast.Ident); ok {
		if id.Name == "_" {
			return lhsTarget{kind: lhsBlank, node: e}
		}
		if define {
			if v, ok := fl.info.Defs[id].(*types.Var); ok {
				loc := fl.local(v, ir.LVar)
				return lhsTarget{kind: lhsNew, local: loc, t: loc.Type, node: e}
			}
		}
	}
	if ix, ok := e.(*ast.IndexExpr); ok {
		if mt, ok := fl.info.TypeOf(ix.X).Underlying().(*types.Map); ok {
			m := fl.expr(ix.X)
			k := fl.exprTo(ix.Index, fl.typ(mt.Key()))
			if snap {
				m, k = fl.snap(m), fl.snap(k)
			}
			return lhsTarget{kind: lhsMap, m: m, k: k, t: fl.typ(mt.Elem()), node: e}
		}
	}
	p := fl.place(e, true)
	if snap {
		p = fl.snapPlace(p)
	}
	return lhsTarget{kind: lhsPlace, place: p, t: p.Type, node: e}
}

// snapPlace snapshots live-local operands of a place.
func (fl *fnLowerer) snapPlace(p *ir.Place) *ir.Place {
	np := &ir.Place{Root: p.Root, Type: p.Type}
	switch r := p.Root.(type) {
	case ir.DerefRoot:
		np.Root = ir.DerefRoot{Ptr: fl.snap(r.Ptr)}
	case ir.SliceRoot:
		np.Root = ir.SliceRoot{Slice: fl.snap(r.Slice), Index: fl.snap(r.Index)}
	}
	for _, pr := range p.Path {
		if pr.Index != nil {
			pr.Index = fl.snap(pr.Index)
		}
		np.Path = append(np.Path, pr)
	}
	return np
}

func (fl *fnLowerer) store(t lhsTarget, v ir.Value) {
	switch t.kind {
	case lhsBlank:
	case lhsNew:
		fl.emit(&ir.DeclVar{At: at(t.node), L: t.local, Init: fl.conv(v, t.t, t.node)})
	case lhsPlace:
		fl.emit(&ir.Store{At: at(t.node), Place: t.place, V: fl.conv(v, t.t, t.node)})
	case lhsMap:
		fl.emit(&ir.MapStore{At: at(t.node), M: t.m, K: t.k, V: fl.conv(v, t.t, t.node)})
	}
}

func (fl *fnLowerer) assign(s *ast.AssignStmt) {
	if op, ok := assignOps[s.Tok]; ok {
		rhs := s.Rhs[0]
		lt := fl.typeOf(s.Lhs[0])
		fl.opAssign(s, s.Lhs[0], op, func() ir.Value {
			if op == token.SHL || op == token.SHR {
				return fl.expr(rhs)
			}
			return fl.exprTo(rhs, lt)
		})
		return
	}
	define := s.Tok == token.DEFINE
	if len(s.Lhs) == len(s.Rhs) {
		multi := len(s.Lhs) > 1
		targets := make([]lhsTarget, len(s.Lhs))
		for i, e := range s.Lhs {
			targets[i] = fl.lhs(e, define, multi)
		}
		vals := make([]ir.Value, len(s.Rhs))
		for i, e := range s.Rhs {
			var v ir.Value
			if targets[i].kind == lhsBlank {
				v = fl.expr(e)
			} else {
				v = fl.exprTo(e, targets[i].t)
			}
			if multi {
				v = fl.snap(v)
			}
			vals[i] = v
		}
		for i := range targets {
			fl.store(targets[i], vals[i])
		}
		return
	}
	targets := make([]lhsTarget, len(s.Lhs))
	for i, e := range s.Lhs {
		targets[i] = fl.lhs(e, define, true)
	}
	vals := fl.exprN1(s.Rhs[0], len(s.Lhs))
	if len(vals) != len(targets) {
		fail(s.Pos(), "assignment count mismatch")
	}
	for i := range targets {
		fl.store(targets[i], vals[i])
	}
}

// opAssign lowers x op= y with x's operands evaluated once.
func (fl *fnLowerer) opAssign(s ast.Node, lhs ast.Expr, op token.Token, rhs func() ir.Value) {
	t := fl.typeOf(lhs)
	k := binOps[op]
	tgt := fl.lhs(lhs, false, false)
	switch tgt.kind {
	case lhsMap:
		y := rhs()
		old := fl.temp(t)
		fl.emit(&ir.MapLookup{At: at(s), Dst: old, M: tgt.m, K: tgt.k})
		r := fl.temp(t)
		fl.emit(&ir.BinOp{At: at(s), Dst: r, Op: k, X: old, Y: y})
		fl.emit(&ir.MapStore{At: at(s), M: tgt.m, K: tgt.k, V: r})
	case lhsPlace:
		y := rhs()
		old := fl.load(tgt.place, s)
		r := fl.temp(t)
		fl.emit(&ir.BinOp{At: at(s), Dst: r, Op: k, X: old, Y: y})
		fl.emit(&ir.Store{At: at(s), Place: tgt.place, V: r})
	default:
		fail(s.Pos(), "invalid compound assignment target")
	}
}

func (fl *fnLowerer) intConst(n int64) *ir.Const {
	return &ir.Const{Type: fl.ts().IntT(), Val: constant.MakeInt64(n)}
}

// rangeStmt lowers range loops over integers, slices, arrays, pointers to
// arrays, strings, and maps.
func (fl *fnLowerer) rangeStmt(s *ast.RangeStmt) {
	label := fl.takeLabel()
	xt := fl.info.TypeOf(s.X)
	head := fl.f.NewBlock("range.head")
	body := fl.f.NewBlock("range.body")
	post := fl.f.NewBlock("range.post")
	done := fl.f.NewBlock("range.done")
	define := s.Tok == token.DEFINE
	key, val := s.Key, s.Value
	if key != nil {
		if id, ok := key.(*ast.Ident); ok && id.Name == "_" {
			key = nil
		}
	}
	if val != nil {
		if id, ok := val.(*ast.Ident); ok && id.Name == "_" {
			val = nil
		}
	}
	bind := func(e ast.Expr, v ir.Value) {
		if e == nil {
			return
		}
		fl.store(fl.lhs(e, define, false), v)
	}
	loop := func(prepare func(), headCond func(), bodyBind func(), step func()) {
		prepare()
		fl.enter(head)
		headCond()
		fl.b = body
		bodyBind()
		fl.targets = append(fl.targets, target{label: label, brk: done, cont: post})
		fl.block(s.Body.List)
		fl.targets = fl.targets[:len(fl.targets)-1]
		fl.enter(post)
		step()
		fl.jump(head)
		fl.b = done
	}
	switch u := xt.Underlying().(type) {
	case *types.Basic:
		if u.Info()&types.IsString != 0 {
			var str ir.Value
			it := fl.temp(fl.ts().IntT())
			n := fl.temp(fl.ts().IntT())
			r := fl.temp(fl.ts().Basic(types.Int32))
			w := fl.temp(fl.ts().IntT())
			loop(func() {
				str = fl.snap(fl.expr(s.X))
				fl.emit(&ir.Len{At: at(s), Dst: n, X: str})
				fl.emit(&ir.Assign{Dst: it, Src: fl.intConst(0)})
			}, func() {
				c := fl.temp(fl.ts().Bool())
				fl.emit(&ir.BinOp{Dst: c, Op: ir.Lt, X: it, Y: n})
				fl.term(&ir.If{At: at(s), Cond: c, Then: body, Else: done})
			}, func() {
				fl.emit(&ir.DecodeRune{At: at(s), Rune: r, Width: w, S: str, I: it})
				bind(key, fl.copyOf(it))
				bind(val, fl.copyOf(r))
			}, func() {
				fl.emit(&ir.BinOp{Dst: it, Op: ir.Add, X: it, Y: w})
			})
			return
		}
		// Range over an integer.
		t := fl.typ(xt)
		it := fl.temp(t)
		var n ir.Value
		loop(func() {
			n = fl.snap(fl.exprTo(s.X, t))
			fl.emit(&ir.Assign{Dst: it, Src: &ir.Const{Type: t, Val: constant.MakeInt64(0)}})
		}, func() {
			c := fl.temp(fl.ts().Bool())
			fl.emit(&ir.BinOp{Dst: c, Op: ir.Lt, X: it, Y: n})
			fl.term(&ir.If{At: at(s), Cond: c, Then: body, Else: done})
		}, func() {
			bind(key, fl.copyOf(it))
		}, func() {
			fl.emit(&ir.BinOp{Dst: it, Op: ir.Add, X: it, Y: &ir.Const{Type: t, Val: constant.MakeInt64(1)}})
		})
		return
	case *types.Chan:
		ct := fl.typ(xt)
		var ch ir.Value
		v := fl.temp(ct.U().Elem)
		ok := fl.temp(fl.ts().Bool())
		loop(func() {
			ch = fl.snap(fl.expr(s.X))
		}, func() {
			fl.emit(&ir.Recv{At: at(s), Dst: v, Ok: ok, Ch: ch})
			fl.term(&ir.If{At: at(s), Cond: ok, Then: body, Else: done})
		}, func() {
			bind(key, v)
		}, func() {})
		return
	case *types.Map:
		mt := fl.typ(xt)
		iter := fl.temp(fl.ts().MapIter(mt))
		ok := fl.temp(fl.ts().Bool())
		var k, v *ir.Local
		if key != nil {
			k = fl.temp(mt.U().Key)
		}
		if val != nil {
			v = fl.temp(mt.U().Elem)
		}
		loop(func() {
			m := fl.expr(s.X)
			fl.emit(&ir.MapIterInit{At: at(s), Iter: iter, M: m})
		}, func() {
			fl.emit(&ir.MapIterNext{At: at(s), Ok: ok, Key: k, Val: v, Iter: iter})
			fl.term(&ir.If{At: at(s), Cond: ok, Then: body, Else: done})
		}, func() {
			if k != nil {
				bind(key, k)
			}
			if v != nil {
				bind(val, v)
			}
		}, func() {})
		return
	}
	// Slices, arrays, and pointers to arrays.
	it := fl.temp(fl.ts().IntT())
	n := fl.temp(fl.ts().IntT())
	var elem func() *ir.Place
	loop(func() {
		switch u := xt.Underlying().(type) {
		case *types.Slice:
			sv := fl.snap(fl.expr(s.X))
			fl.emit(&ir.Len{At: at(s), Dst: n, X: sv})
			et := fl.typ(u.Elem())
			elem = func() *ir.Place { return &ir.Place{Root: ir.SliceRoot{Slice: sv, Index: it}, Type: et} }
		case *types.Array:
			fl.emit(&ir.Assign{Dst: n, Src: fl.intConst(u.Len())})
			et := fl.typ(u.Elem())
			if val != nil {
				arr := fl.expr(s.X)
				elem = func() *ir.Place {
					return &ir.Place{Root: ir.ValueRoot{Value: arr}, Path: []ir.Proj{{Index: it, Type: et}}, Type: et}
				}
			} else if !isConstLen(fl.info, s.X) {
				fl.expr(s.X)
			}
		case *types.Pointer:
			at_ := u.Elem().Underlying().(*types.Array)
			fl.emit(&ir.Assign{Dst: n, Src: fl.intConst(at_.Len())})
			et := fl.typ(at_.Elem())
			pv := fl.snap(fl.expr(s.X))
			elem = func() *ir.Place {
				return &ir.Place{Root: ir.DerefRoot{Ptr: pv}, Path: []ir.Proj{{Index: it, Type: et}}, Type: et}
			}
		default:
			fail(s.X.Pos(), "unsupported range operand %s", xt)
		}
		fl.emit(&ir.Assign{Dst: it, Src: fl.intConst(0)})
	}, func() {
		c := fl.temp(fl.ts().Bool())
		fl.emit(&ir.BinOp{Dst: c, Op: ir.Lt, X: it, Y: n})
		fl.term(&ir.If{At: at(s), Cond: c, Then: body, Else: done})
	}, func() {
		bind(key, fl.copyOf(it))
		if val != nil {
			bind(val, fl.load(elem(), s))
		}
	}, func() {
		fl.emit(&ir.BinOp{Dst: it, Op: ir.Add, X: it, Y: fl.intConst(1)})
	})
}

func isConstLen(info *types.Info, e ast.Expr) bool {
	_, isCall := ast.Unparen(e).(*ast.CallExpr)
	return !isCall
}

// copyOf snapshots a temp that the loop machinery will keep mutating.
func (fl *fnLowerer) copyOf(v *ir.Local) ir.Value {
	t := fl.temp(v.Type)
	fl.emit(&ir.Assign{Dst: t, Src: v})
	return t
}

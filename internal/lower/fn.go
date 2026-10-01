package lower

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/eugenioenko/goalchemy/internal/ir"

	"golang.org/x/tools/go/packages"
)

type target struct {
	label string
	brk   *ir.Block
	cont  *ir.Block // nil for switch
}

type fnLowerer struct {
	l       *Lowerer
	p       *packages.Package
	info    *types.Info
	f       *ir.Func
	b       *ir.Block // current block; nil when unreachable
	locals  map[*types.Var]*ir.Local
	targets []target
	// pendingLabel names the label attached to the next loop or switch.
	pendingLabel string
	// fallTo is the body of the next switch clause for fallthrough.
	fallTo *ir.Block
	litN   int
}

func (l *Lowerer) newFn(p *packages.Package, f *ir.Func) *fnLowerer {
	fl := &fnLowerer{l: l, f: f, locals: map[*types.Var]*ir.Local{}}
	if p != nil {
		fl.setPkg(p)
	}
	return fl
}

func (fl *fnLowerer) setPkg(p *packages.Package) {
	fl.p, fl.info = p, p.TypesInfo
}

func (fl *fnLowerer) ts() *ir.Types { return fl.l.ts }

func (fl *fnLowerer) start() {
	fl.b = fl.f.NewBlock("entry")
}

// cur returns the current block, creating a fresh unreachable block when
// control flow cannot reach this point.
func (fl *fnLowerer) cur() *ir.Block {
	if fl.b == nil {
		fl.b = fl.f.NewBlock("unreachable")
	}
	return fl.b
}

func (fl *fnLowerer) emit(i ir.Instr) {
	b := fl.cur()
	b.Instrs = append(b.Instrs, i)
}

func (fl *fnLowerer) term(t ir.Terminator) {
	b := fl.cur()
	b.Term = t
	fl.b = nil
}

func (fl *fnLowerer) jump(to *ir.Block) {
	if fl.b != nil {
		fl.term(&ir.Jump{Target: to})
	}
}

func (fl *fnLowerer) enter(b *ir.Block) {
	fl.jump(b)
	fl.b = b
}

func (fl *fnLowerer) temp(t *ir.Type) *ir.Local {
	return fl.f.NewLocal("", t, ir.LTemp)
}

func (fl *fnLowerer) typ(t types.Type) *ir.Type { return fl.l.ts.Of(t) }

func (fl *fnLowerer) typeOf(e ast.Expr) *ir.Type {
	t := fl.info.TypeOf(e)
	if t == nil {
		fail(e.Pos(), "expression has no type")
	}
	return fl.typ(t)
}

func at(n ast.Node) ir.At {
	if n == nil {
		return ir.At{}
	}
	return ir.At{Pos: n.Pos()}
}

// local returns the IR local for a source variable, creating it on first use.
func (fl *fnLowerer) local(v *types.Var, kind ir.LocalKind) *ir.Local {
	if loc, ok := fl.locals[v]; ok {
		return loc
	}
	loc := fl.f.NewLocal(v.Name(), fl.typ(v.Type()), kind)
	loc.Boxed = fl.l.boxed[v]
	loc.Pos = v.Pos()
	fl.locals[v] = loc
	return loc
}

func (fl *fnLowerer) params(list *ast.FieldList, vars []*types.Var, kind ir.LocalKind) {
	for _, v := range vars {
		name := v.Name()
		var loc *ir.Local
		if name == "" || name == "_" {
			loc = fl.f.NewLocal("", fl.typ(v.Type()), kind)
		} else {
			loc = fl.local(v, kind)
		}
		fl.f.Params = append(fl.f.Params, loc)
	}
}

func (fl *fnLowerer) results(list *ast.FieldList, res *types.Tuple) {
	for i := 0; i < res.Len(); i++ {
		v := res.At(i)
		var loc *ir.Local
		if v.Name() == "" || v.Name() == "_" {
			loc = fl.f.NewLocal("", fl.typ(v.Type()), ir.LResult)
		} else {
			loc = fl.local(v, ir.LResult)
		}
		fl.f.Results = append(fl.f.Results, loc)
	}
}

// body lowers a function body. Results are zero-initialized at entry.
func (fl *fnLowerer) body(b *ast.BlockStmt) {
	fl.start()
	for _, r := range fl.f.Results {
		fl.emit(&ir.DeclVar{L: r})
	}
	for _, p := range fl.f.Params {
		if p.Boxed {
			fl.emit(&ir.BoxParam{L: p})
		}
	}
	fl.block(b.List)
	fl.finish()
}

func (fl *fnLowerer) finish() {
	if fl.b != nil {
		if len(fl.f.Results) == 0 {
			fl.term(&ir.Return{})
		} else {
			fl.term(&ir.Unreachable{})
		}
	}
	for _, b := range fl.f.Blocks {
		if b.Term == nil {
			b.Term = &ir.Unreachable{}
		}
	}
}

func (fl *fnLowerer) block(list []ast.Stmt) {
	for _, s := range list {
		fl.stmt(s)
	}
}

func (fl *fnLowerer) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.EmptyStmt:
	case *ast.BlockStmt:
		fl.block(s.List)
	case *ast.ExprStmt:
		fl.exprN(s.X)
	case *ast.DeclStmt:
		fl.declStmt(s)
	case *ast.AssignStmt:
		fl.assign(s)
	case *ast.IncDecStmt:
		op := token.ADD
		if s.Tok == token.DEC {
			op = token.SUB
		}
		one := &ir.Const{Type: fl.typeOf(s.X), Val: constant.MakeInt64(1)}
		fl.opAssign(s, s.X, op, func() ir.Value { return one })
	case *ast.ReturnStmt:
		fl.returnStmt(s)
	case *ast.IfStmt:
		fl.ifStmt(s)
	case *ast.ForStmt:
		fl.forStmt(s)
	case *ast.RangeStmt:
		fl.rangeStmt(s)
	case *ast.SwitchStmt:
		fl.switchStmt(s)
	case *ast.TypeSwitchStmt:
		fl.typeSwitch(s)
	case *ast.LabeledStmt:
		fl.pendingLabel = s.Label.Name
		fl.stmt(s.Stmt)
		fl.pendingLabel = ""
	case *ast.BranchStmt:
		fl.branch(s)
	case *ast.DeferStmt:
		fl.deferStmt(s)
	default:
		fl.cooperativeStmt(s)
	}
}

func (fl *fnLowerer) takeLabel() string {
	l := fl.pendingLabel
	fl.pendingLabel = ""
	return l
}

func (fl *fnLowerer) declStmt(s *ast.DeclStmt) {
	gd, ok := s.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return
	}
	for _, sp := range gd.Specs {
		vs := sp.(*ast.ValueSpec)
		var vals []ir.Value
		switch {
		case len(vs.Values) == len(vs.Names):
			for i, e := range vs.Values {
				vals = append(vals, fl.exprTo(e, fl.typ(fl.info.Defs[vs.Names[i]].Type())))
			}
		case len(vs.Values) == 1:
			vals = fl.exprN(vs.Values[0])
		}
		for i, n := range vs.Names {
			v, _ := fl.info.Defs[n].(*types.Var)
			if v == nil || n.Name == "_" {
				continue
			}
			loc := fl.local(v, ir.LVar)
			var init ir.Value
			if vals != nil {
				init = fl.conv(vals[i], loc.Type, n)
			}
			fl.emit(&ir.DeclVar{At: at(n), L: loc, Init: init})
		}
	}
}

func (fl *fnLowerer) returnStmt(s *ast.ReturnStmt) {
	res := fl.f.Results
	var vals []ir.Value
	if len(s.Results) == 1 && len(res) > 1 {
		vals = fl.exprN(s.Results[0])
	} else {
		for i, e := range s.Results {
			vals = append(vals, fl.exprTo(e, res[i].Type))
		}
	}
	if len(vals) > 1 {
		for i := range vals {
			vals[i] = fl.snap(vals[i])
		}
	}
	for i, v := range vals {
		fl.emit(&ir.Store{At: at(s), Place: localPlace(res[i]), V: fl.conv(v, res[i].Type, s)})
	}
	fl.term(&ir.Return{At: at(s)})
}

func localPlace(l *ir.Local) *ir.Place {
	return &ir.Place{Root: ir.LocalRoot{Local: l}, Type: l.Type}
}

func (fl *fnLowerer) ifStmt(s *ast.IfStmt) {
	if s.Init != nil {
		fl.stmt(s.Init)
	}
	then := fl.f.NewBlock("if.then")
	done := fl.f.NewBlock("if.done")
	els := done
	if s.Else != nil {
		els = fl.f.NewBlock("if.else")
	}
	fl.cond(s.Cond, then, els)
	fl.b = then
	fl.block(s.Body.List)
	fl.jump(done)
	if s.Else != nil {
		fl.b = els
		fl.stmt(s.Else)
		fl.jump(done)
	}
	fl.b = done
}

// cond branches on a Boolean expression with short-circuit evaluation.
func (fl *fnLowerer) cond(e ast.Expr, t, f *ir.Block) {
	switch x := e.(type) {
	case *ast.ParenExpr:
		fl.cond(x.X, t, f)
		return
	case *ast.UnaryExpr:
		if x.Op == token.NOT {
			fl.cond(x.X, f, t)
			return
		}
	case *ast.BinaryExpr:
		switch x.Op {
		case token.LAND:
			mid := fl.f.NewBlock("and.rhs")
			fl.cond(x.X, mid, f)
			fl.b = mid
			fl.cond(x.Y, t, f)
			return
		case token.LOR:
			mid := fl.f.NewBlock("or.rhs")
			fl.cond(x.X, t, mid)
			fl.b = mid
			fl.cond(x.Y, t, f)
			return
		}
	}
	v := fl.expr(e)
	fl.term(&ir.If{At: at(e), Cond: v, Then: t, Else: f})
}

func (fl *fnLowerer) loopVarsOf(init ast.Stmt) []*ir.Local {
	as, ok := init.(*ast.AssignStmt)
	if !ok || as.Tok != token.DEFINE {
		return nil
	}
	var out []*ir.Local
	for _, lhs := range as.Lhs {
		if id, ok := lhs.(*ast.Ident); ok {
			if v, ok := fl.info.Defs[id].(*types.Var); ok {
				if loc := fl.locals[v]; loc != nil && loc.Boxed {
					out = append(out, loc)
				}
			}
		}
	}
	return out
}

func (fl *fnLowerer) forStmt(s *ast.ForStmt) {
	label := fl.takeLabel()
	if s.Init != nil {
		fl.stmt(s.Init)
	}
	perIter := fl.loopVarsOf(s.Init)
	head := fl.f.NewBlock("for.head")
	body := fl.f.NewBlock("for.body")
	post := fl.f.NewBlock("for.post")
	done := fl.f.NewBlock("for.done")
	fl.enter(head)
	if s.Cond != nil {
		fl.cond(s.Cond, body, done)
	} else {
		fl.jump(body)
	}
	fl.b = body
	fl.targets = append(fl.targets, target{label: label, brk: done, cont: post})
	fl.block(s.Body.List)
	fl.targets = fl.targets[:len(fl.targets)-1]
	fl.enter(post)
	for _, v := range perIter {
		fl.emit(&ir.Rebind{At: at(s), L: v})
	}
	if s.Post != nil {
		fl.stmt(s.Post)
	}
	fl.jump(head)
	fl.b = done
}

func (fl *fnLowerer) branch(s *ast.BranchStmt) {
	switch s.Tok {
	case token.FALLTHROUGH:
		if fl.fallTo == nil {
			fail(s.Pos(), "misplaced fallthrough")
		}
		fl.jump(fl.fallTo)
		return
	case token.GOTO:
		fail(s.Pos(), "goto is not supported")
	}
	for i := len(fl.targets) - 1; i >= 0; i-- {
		t := fl.targets[i]
		if s.Label != nil && t.label != s.Label.Name {
			continue
		}
		if s.Tok == token.BREAK {
			fl.jump(t.brk)
			return
		}
		if t.cont != nil {
			fl.jump(t.cont)
			return
		}
		if s.Label != nil {
			break
		}
	}
	fail(s.Pos(), "branch target not found")
}

func (fl *fnLowerer) switchStmt(s *ast.SwitchStmt) {
	label := fl.takeLabel()
	if s.Init != nil {
		fl.stmt(s.Init)
	}
	var tag ir.Value
	var tagT *ir.Type
	if s.Tag != nil {
		tagT = fl.typeOf(s.Tag)
		tag = fl.snap(fl.expr(s.Tag))
	}
	done := fl.f.NewBlock("switch.done")
	clauses := s.Body.List
	bodies := make([]*ir.Block, len(clauses))
	for i := range clauses {
		bodies[i] = fl.f.NewBlock("switch.case")
	}
	var deflt *ir.Block
	for i, c := range clauses {
		cc := c.(*ast.CaseClause)
		if cc.List == nil {
			deflt = bodies[i]
			continue
		}
		for _, e := range cc.List {
			next := fl.f.NewBlock("switch.next")
			if tag == nil {
				fl.cond(e, bodies[i], next)
			} else {
				v := fl.expr(e)
				eq := fl.equal(tag, tagT, v, fl.typeOf(e), e)
				fl.term(&ir.If{At: at(e), Cond: eq, Then: bodies[i], Else: next})
			}
			fl.b = next
		}
	}
	if deflt != nil {
		fl.jump(deflt)
	} else {
		fl.jump(done)
	}
	fl.targets = append(fl.targets, target{label: label, brk: done})
	for i, c := range clauses {
		cc := c.(*ast.CaseClause)
		fl.b = bodies[i]
		saved := fl.fallTo
		fl.fallTo = nil
		if i+1 < len(bodies) {
			fl.fallTo = bodies[i+1]
		}
		fl.block(cc.Body)
		fl.fallTo = saved
		fl.jump(done)
	}
	fl.targets = fl.targets[:len(fl.targets)-1]
	fl.b = done
}

func (fl *fnLowerer) typeSwitch(s *ast.TypeSwitchStmt) {
	label := fl.takeLabel()
	if s.Init != nil {
		fl.stmt(s.Init)
	}
	var x ast.Expr
	switch a := s.Assign.(type) {
	case *ast.AssignStmt:
		x = a.Rhs[0].(*ast.TypeAssertExpr).X
	case *ast.ExprStmt:
		x = a.X.(*ast.TypeAssertExpr).X
	}
	xt := fl.typeOf(x)
	xv := fl.snap(fl.expr(x))
	done := fl.f.NewBlock("tswitch.done")
	clauses := s.Body.List
	bodies := make([]*ir.Block, len(clauses))
	for i := range clauses {
		bodies[i] = fl.f.NewBlock("tswitch.case")
	}
	var deflt *ir.Block
	deflIdx := -1
	for i, c := range clauses {
		cc := c.(*ast.CaseClause)
		obj, _ := fl.info.Implicits[cc].(*types.Var)
		if cc.List == nil {
			deflt, deflIdx = bodies[i], i
			continue
		}
		for _, e := range cc.List {
			next := fl.f.NewBlock("tswitch.next")
			tv := fl.info.Types[e]
			if tv.IsNil() {
				isNil := fl.temp(fl.ts().Bool())
				fl.emit(&ir.BinOp{At: at(e), Dst: isNil, Op: ir.Eq, X: xv, Y: &ir.Const{Type: xt, Nil: true}})
				fl.term(&ir.If{At: at(e), Cond: isNil, Then: fl.bindCase(obj, cc, xv, nil, bodies[i]), Else: next})
			} else {
				ct := fl.typ(tv.Type)
				val := fl.temp(ct)
				ok := fl.temp(fl.ts().Bool())
				fl.emit(&ir.TypeAssert{At: at(e), Dst: val, Ok: ok, X: xv, T: ct})
				var bound ir.Value = xv
				if len(cc.List) == 1 {
					bound = val
				}
				fl.term(&ir.If{At: at(e), Cond: ok, Then: fl.bindCase(obj, cc, bound, nil, bodies[i]), Else: next})
			}
			fl.b = next
		}
	}
	if deflt != nil {
		cc := clauses[deflIdx].(*ast.CaseClause)
		obj, _ := fl.info.Implicits[cc].(*types.Var)
		fl.jump(fl.bindCase(obj, cc, xv, nil, deflt))
	} else {
		fl.jump(done)
	}
	fl.targets = append(fl.targets, target{label: label, brk: done})
	for i, c := range clauses {
		cc := c.(*ast.CaseClause)
		fl.b = bodies[i]
		fl.block(cc.Body)
		fl.jump(done)
	}
	fl.targets = fl.targets[:len(fl.targets)-1]
	fl.b = done
}

// bindCase returns a block that declares the clause variable and continues
// to body. Without a variable it returns body itself.
func (fl *fnLowerer) bindCase(obj *types.Var, cc *ast.CaseClause, v ir.Value, _ *ir.Type, body *ir.Block) *ir.Block {
	if obj == nil || obj.Name() == "_" {
		return body
	}
	b := fl.f.NewBlock("tswitch.bind")
	saved := fl.b
	fl.b = b
	loc := fl.local(obj, ir.LVar)
	fl.emit(&ir.DeclVar{At: at(cc), L: loc, Init: fl.conv(v, loc.Type, cc)})
	fl.jump(body)
	fl.b = saved
	return b
}

func (fl *fnLowerer) initializer(in *types.Initializer) {
	if len(in.Lhs) == 1 {
		v := in.Lhs[0]
		if g, ok := fl.l.globals[v]; ok {
			val := fl.exprTo(in.Rhs, g.Type)
			fl.emit(&ir.Store{At: at(in.Rhs), Place: &ir.Place{Root: ir.GlobalRoot{Global: g}, Type: g.Type}, V: val})
		} else {
			fl.exprN(in.Rhs)
		}
		return
	}
	vals := fl.exprN(in.Rhs)
	for i, v := range in.Lhs {
		if g, ok := fl.l.globals[v]; ok {
			fl.emit(&ir.Store{At: at(in.Rhs), Place: &ir.Place{Root: ir.GlobalRoot{Global: g}, Type: g.Type}, V: fl.conv(vals[i], g.Type, in.Rhs)})
		}
	}
}

func (fl *fnLowerer) cooperativeStmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.SendStmt:
		ch := fl.expr(s.Chan)
		v := fl.exprTo(s.Value, ch.IRType().U().Elem)
		fl.emit(&ir.Send{At: at(s), Ch: ch, V: v})
	case *ast.GoStmt:
		fl.goStmt(s)
	case *ast.SelectStmt:
		fl.selectStmt(s)
	default:
		fail(s.Pos(), "unsupported statement %T", s)
	}
}

func (fl *fnLowerer) goStmt(s *ast.GoStmt) {
	e := s.Call
	fun := ast.Unparen(e.Fun)
	if id, ok := fun.(*ast.Ident); ok {
		if b, ok := fl.info.Uses[id].(*types.Builtin); ok {
			c := fl.builtinCall(e, b.Name(), "go")
			fl.emit(&ir.Go{At: at(s), Call: c})
			return
		}
	}
	c := fl.prepareCall(e, true)
	fl.emit(&ir.Go{At: at(s), Call: c})
}

// selectStmt evaluates every channel operand and send value in source
// order, commits one case with ir.Select, then branches on the chosen index.
func (fl *fnLowerer) selectStmt(s *ast.SelectStmt) {
	label := fl.takeLabel()
	sel := &ir.Select{At: at(s), Index: fl.temp(fl.ts().IntT())}
	type recvBind struct {
		assign *ast.AssignStmt
		dst    *ir.Local
		ok     *ir.Local
	}
	clauses := s.Body.List
	caseOf := make([]int, len(clauses))
	binds := make([]recvBind, len(clauses))
	for i, c := range clauses {
		cc := c.(*ast.CommClause)
		caseOf[i] = -1
		switch comm := cc.Comm.(type) {
		case nil:
			sel.Default = true
		case *ast.SendStmt:
			ch := fl.expr(comm.Chan)
			v := fl.exprTo(comm.Value, ch.IRType().U().Elem)
			caseOf[i] = len(sel.Cases)
			sel.Cases = append(sel.Cases, ir.SelectCase{Send: true, Ch: fl.snap(ch), V: fl.snap(v)})
		case *ast.ExprStmt:
			ch := fl.expr(ast.Unparen(comm.X).(*ast.UnaryExpr).X)
			caseOf[i] = len(sel.Cases)
			sel.Cases = append(sel.Cases, ir.SelectCase{Ch: fl.snap(ch)})
		case *ast.AssignStmt:
			ux := ast.Unparen(comm.Rhs[0]).(*ast.UnaryExpr)
			ch := fl.expr(ux.X)
			sc := ir.SelectCase{Ch: fl.snap(ch), Dst: fl.temp(ch.IRType().U().Elem)}
			if len(comm.Lhs) == 2 {
				sc.Ok = fl.temp(fl.ts().Bool())
			}
			caseOf[i] = len(sel.Cases)
			binds[i] = recvBind{assign: comm, dst: sc.Dst, ok: sc.Ok}
			sel.Cases = append(sel.Cases, sc)
		}
	}
	fl.emit(sel)
	done := fl.f.NewBlock("select.done")
	fl.targets = append(fl.targets, target{label: label, brk: done})
	for i, c := range clauses {
		cc := c.(*ast.CommClause)
		body := fl.f.NewBlock("select.case")
		next := fl.f.NewBlock("select.next")
		want := int64(caseOf[i])
		cond := fl.temp(fl.ts().Bool())
		fl.emit(&ir.BinOp{At: at(cc), Dst: cond, Op: ir.Eq, X: sel.Index, Y: fl.intConst(want)})
		fl.term(&ir.If{At: at(cc), Cond: cond, Then: body, Else: next})
		fl.b = body
		if b := binds[i]; b.assign != nil {
			define := b.assign.Tok == token.DEFINE
			vals := []ir.Value{b.dst}
			if b.ok != nil {
				vals = append(vals, b.ok)
			}
			for j, lhs := range b.assign.Lhs {
				fl.store(fl.lhs(lhs, define, false), vals[j])
			}
		}
		fl.block(cc.Body)
		fl.jump(done)
		fl.b = next
	}
	fl.jump(done)
	fl.targets = fl.targets[:len(fl.targets)-1]
	fl.b = done
}

package lower

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"github.com/eugenioenko/goalchemy/internal/jsontype"
)

// jsonMarshal lowers json.Marshal and json.MarshalIndent. Operands of a
// static non-interface type are encoded through a generated descriptor of
// that type; interface operands are encoded from their dynamic value.
func (fl *fnLowerer) jsonMarshal(e *ast.CallExpr, obj types.Object, name string) []ir.Value {
	pkg := obj.Pkg()
	if pkg.Path() != catalog.JSONPackage {
		fl.term(&ir.PanicRuntime{At: at(e), Msg: "json: the native encoder is unavailable"})
		var zero []ir.Value
		res := obj.(*types.Func).Signature().Results()
		for i := 0; i < res.Len(); i++ {
			zero = append(zero, fl.zeroValue(fl.typ(res.At(i).Type())))
		}
		return zero
	}
	g := &jsonGen{l: fl.l, pkg: pkg}
	anyT := fl.ts().Any()
	arg := e.Args[0]
	tv := fl.info.Types[arg]
	var args []ir.Value
	helper := "marshalAny"
	if tv.IsNil() || types.IsInterface(tv.Type) {
		args = append(args, fl.exprTo(arg, anyT))
	} else {
		t := fl.typ(tv.Type)
		v := fl.exprTo(arg, t)
		p := fl.temp(fl.ts().PointerTo(t))
		fl.emit(&ir.New{At: at(e), Dst: p})
		fl.emit(&ir.Store{At: at(e), Place: &ir.Place{Root: ir.DerefRoot{Ptr: p}, Type: t}, V: v})
		args = append(args, fl.conv(p, anyT, e))
		helper = "marshalTyped"
	}
	if name == "MarshalIndent" {
		helper = map[string]string{"marshalAny": "marshalIndentAny", "marshalTyped": "marshalIndentTyped"}[helper]
		args = append(args, fl.exprTo(e.Args[1], fl.ts().String()), fl.exprTo(e.Args[2], fl.ts().String()))
	}
	if helper == "marshalTyped" || helper == "marshalIndentTyped" {
		desc := g.desc(tv.Type)
		d := fl.temp(desc.Sig.U().Results[0])
		fl.emit(&ir.Call{At: at(e), Kind: ir.CallStatic, Func: desc, Dsts: []*ir.Local{d}})
		args = append(args[:1], append([]ir.Value{d}, args[1:]...)...)
	}
	f := g.helper(helper)
	sig := f.Sig.U()
	var out []ir.Value
	c := &ir.Call{At: at(e), Kind: ir.CallStatic, Func: f, Args: args}
	for _, r := range sig.Results {
		d := fl.temp(r)
		c.Dsts = append(c.Dsts, d)
		out = append(out, d)
	}
	fl.emit(c)
	return out
}

type jsonGen struct {
	l   *Lowerer
	pkg *types.Package
}

func (g *jsonGen) helper(name string) *ir.Func {
	obj, ok := g.pkg.Scope().Lookup(name).(*types.Func)
	if !ok {
		fail(token.NoPos, "json.%s is missing", name)
	}
	f := g.l.funcs[obj]
	if f == nil {
		fail(token.NoPos, "json.%s is not lowered", name)
	}
	return f
}

func (g *jsonGen) paramType(helper string, i int) *ir.Type {
	obj := g.pkg.Scope().Lookup(helper).(*types.Func)
	return g.l.ts.Of(obj.Signature().Params().At(i).Type())
}

func (g *jsonGen) typeID(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() })
}

func strConst(ts *ir.Types, s string) ir.Value {
	return &ir.Const{Type: ts.String(), Val: constant.MakeString(s)}
}

func intConst(ts *ir.Types, n int64) ir.Value {
	return &ir.Const{Type: ts.IntT(), Val: constant.MakeInt64(n)}
}

func boolConst(ts *ir.Types, b bool) ir.Value {
	return &ir.Const{Type: ts.Bool(), Val: constant.MakeBool(b)}
}

// desc returns the function that builds, once, the std/encoding/json
// descriptor of t. The descriptor is cached in a global before its children
// are described, so recursive types terminate.
func (g *jsonGen) desc(t types.Type) *ir.Func {
	l := g.l
	key := "json.desc|" + g.typeID(t)
	if f, ok := l.wrappers[key]; ok {
		return f
	}
	ts := l.ts
	newType := g.helper("newType")
	infoT := newType.Sig.U().Results[0]
	sig := ts.Of(types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", infoT.Go)), false))
	base := sanitize(jsontype.TypeString(t))
	f := l.addFunc(&ir.Func{Name: "json.describe[" + jsontype.TypeString(t) + "]", Sym: l.sym("json_desc_" + base),
		Sig: sig, Pkg: g.pkg.Path(), Wrapper: true})
	l.wrappers[key] = f
	cache := &ir.Global{ID: len(l.out.Globals), Name: "desc_" + base, Pkg: g.pkg.Path(), Sym: l.sym("json_cache_" + base), Type: infoT}
	l.out.Globals = append(l.out.Globals, cache)

	fl := l.newFn(nil, f)
	fl.start()
	result := f.NewLocal("", infoT, ir.LResult)
	f.Results = append(f.Results, result)
	fl.emit(&ir.DeclVar{L: result})
	cachePlace := &ir.Place{Root: ir.GlobalRoot{Global: cache}, Type: infoT}
	cached := fl.load(cachePlace, nil)
	missing := fl.temp(ts.Bool())
	fl.emit(&ir.BinOp{Dst: missing, Op: ir.Eq, X: cached, Y: &ir.Const{Type: infoT, Nil: true}})
	hit := f.NewBlock("hit")
	build := f.NewBlock("build")
	fl.term(&ir.If{Cond: missing, Then: build, Else: hit})
	fl.b = hit
	fl.emit(&ir.Store{Place: localPlace(result), V: cached})
	fl.term(&ir.Return{})
	fl.b = build

	kind := jsontype.KindOf(t)
	flags := jsontype.Flags(t)
	leaf := jsontype.Leaf(t)
	var n int64
	if a, ok := t.Underlying().(*types.Array); ok {
		n = a.Len()
	}
	info := fl.temp(infoT)
	fl.emit(&ir.Call{Kind: ir.CallStatic, Func: newType, Dsts: []*ir.Local{info},
		Args: []ir.Value{strConst(ts, jsontype.TypeString(t)), intConst(ts, int64(kind)), intConst(ts, int64(flags)), intConst(ts, n)}})
	fl.emit(&ir.Store{Place: cachePlace, V: info})

	nilInfo := &ir.Const{Type: infoT, Nil: true}
	elem, keyDesc := ir.Value(nilInfo), ir.Value(nilInfo)
	accessor := func(i int) ir.Value { return &ir.Const{Type: g.paramType("describe", i), Nil: true} }
	load, value, deref, length, index, entries := accessor(3), accessor(4), accessor(5), accessor(6), accessor(7), accessor(8)
	call := func(d *ir.Func) ir.Value {
		dst := fl.temp(infoT)
		fl.emit(&ir.Call{Kind: ir.CallStatic, Func: d, Dsts: []*ir.Local{dst}})
		return dst
	}
	if flags&(jsontype.MarshalJSON|jsontype.MarshalText) != 0 && kind != jsontype.Pointer {
		value = g.accessor(t, "value", g.paramType("describe", 4))
	}
	switch kind {
	case jsontype.Bool, jsontype.Int, jsontype.Uint, jsontype.Float32, jsontype.Float64, jsontype.String, jsontype.Bytes:
		load = g.accessor(t, "load", g.paramType("describe", 3))
	case jsontype.Interface:
		value = g.accessor(t, "value", g.paramType("describe", 4))
	case jsontype.Pointer:
		deref = g.accessor(t, "deref", g.paramType("describe", 5))
		if !leaf {
			elem = call(g.desc(t.Underlying().(*types.Pointer).Elem()))
		}
	case jsontype.Slice:
		length = g.accessor(t, "length", g.paramType("describe", 6))
		if !leaf {
			index = g.accessor(t, "index", g.paramType("describe", 7))
			elem = call(g.desc(t.Underlying().(*types.Slice).Elem()))
		}
	case jsontype.Array:
		if !leaf {
			index = g.accessor(t, "index", g.paramType("describe", 7))
			elem = call(g.desc(t.Underlying().(*types.Array).Elem()))
		}
	case jsontype.Map:
		length = g.accessor(t, "length", g.paramType("describe", 6))
		if !leaf {
			m := t.Underlying().(*types.Map)
			entries = g.accessor(t, "entries", g.paramType("describe", 8))
			keyDesc = call(g.desc(m.Key()))
			elem = call(g.desc(m.Elem()))
		}
	}
	fl.emit(&ir.Call{Kind: ir.CallStatic, Func: g.helper("describe"),
		Args: []ir.Value{info, elem, keyDesc, load, value, deref, length, index, entries}})
	if kind == jsontype.Struct && !leaf {
		addField := g.helper("addField")
		addrT := g.paramType("addField", 6)
		for _, fd := range jsontype.Fields(t) {
			ft := call(g.desc(fd.Type))
			fl.emit(&ir.Call{Kind: ir.CallStatic, Func: addField, Args: []ir.Value{info, strConst(ts, fd.Key),
				boolConst(ts, fd.OmitEmpty), boolConst(ts, fd.Quoted), boolConst(ts, fd.ViaPtr), ft, g.fieldAccessor(t, fd, addrT)}})
		}
	}
	fl.emit(&ir.Store{Place: localPlace(result), V: info})
	fl.term(&ir.Return{})
	fl.finish()
	return f
}

func (g *jsonGen) newAccessor(t types.Type, what string, sig *ir.Type) (*fnLowerer, *ir.Func, *ir.Local) {
	l := g.l
	base := sanitize(jsontype.TypeString(t))
	f := l.addFunc(&ir.Func{Name: "json." + what + "[" + jsontype.TypeString(t) + "]", Sym: l.sym("json_" + what + "_" + base),
		Sig: sig, Pkg: g.pkg.Path(), Wrapper: true})
	fl := l.newFn(nil, f)
	fl.start()
	u := sig.U()
	for i, pt := range u.Params {
		name := "p"
		if i > 0 {
			name = "i"
		}
		f.Params = append(f.Params, f.NewLocal(name, pt, ir.LParam))
	}
	for _, rt := range u.Results {
		r := f.NewLocal("", rt, ir.LResult)
		f.Results = append(f.Results, r)
		fl.emit(&ir.DeclVar{L: r})
	}
	ptrT := l.ts.Of(types.NewPointer(t))
	fl.markBoxed(ptrT)
	ptr := fl.temp(ptrT)
	fl.emit(&ir.TypeAssert{Dst: ptr, X: f.Params[0], T: ptrT})
	return fl, f, ptr
}

func (g *jsonGen) accessor(t types.Type, what string, sig *ir.Type) ir.Value {
	key := "json." + what + "|" + g.typeID(t)
	if f, ok := g.l.wrappers[key]; ok {
		return &ir.FuncRef{Func: f, Type: sig}
	}
	fl, f, ptr := g.newAccessor(t, what, sig)
	g.l.wrappers[key] = f
	ts := g.l.ts
	tt := ts.Of(t)
	anyT := ts.Any()
	v := fl.load(&ir.Place{Root: ir.DerefRoot{Ptr: ptr}, Type: tt}, nil)
	res := f.Results
	ret := func(x ir.Value) {
		fl.emit(&ir.Store{Place: localPlace(res[0]), V: x})
		fl.term(&ir.Return{})
	}
	isNil := func(x ir.Value) (*ir.Block, *ir.Block) {
		c := fl.temp(ts.Bool())
		fl.emit(&ir.BinOp{Dst: c, Op: ir.Eq, X: x, Y: &ir.Const{Type: x.IRType(), Nil: true}})
		yes, no := f.NewBlock("nil"), f.NewBlock("value")
		fl.term(&ir.If{Cond: c, Then: yes, Else: no})
		return yes, no
	}
	switch what {
	case "load":
		ret(fl.conv(g.canonical(fl, v, t), anyT, nil))
	case "value":
		ret(fl.conv(v, anyT, nil))
	case "deref":
		yes, no := isNil(v)
		fl.b = yes
		fl.term(&ir.Return{})
		fl.b = no
		ret(fl.conv(v, anyT, nil))
	case "length":
		yes, no := isNil(v)
		fl.b = yes
		ret(intConst(ts, -1))
		fl.b = no
		n := fl.temp(ts.IntT())
		fl.emit(&ir.Len{Dst: n, X: v})
		ret(n)
	case "index":
		var el *ir.Local
		switch u := t.Underlying().(type) {
		case *types.Slice:
			el = fl.load(&ir.Place{Root: ir.SliceRoot{Slice: v, Index: f.Params[1]}, Type: ts.Of(u.Elem())}, nil)
		case *types.Array:
			et := ts.Of(u.Elem())
			el = fl.load(&ir.Place{Root: ir.DerefRoot{Ptr: ptr}, Path: []ir.Proj{{Index: f.Params[1], Type: et}}, Type: et}, nil)
		}
		ret(fl.conv(g.copyOf(fl, el), anyT, nil))
	case "entries":
		m := t.Underlying().(*types.Map)
		yes, no := isNil(v)
		fl.b = yes
		fl.emit(&ir.Store{Place: localPlace(res[2]), V: boolConst(ts, true)})
		fl.term(&ir.Return{})
		fl.b = no
		mt := ts.Of(t)
		iter := fl.temp(ts.MapIter(mt))
		ok := fl.temp(ts.Bool())
		k := fl.temp(ts.Of(m.Key()))
		val := fl.temp(ts.Of(m.Elem()))
		fl.emit(&ir.MapIterInit{Iter: iter, M: v})
		head, body, done := f.NewBlock("range.head"), f.NewBlock("range.body"), f.NewBlock("range.done")
		fl.jump(head)
		fl.b = head
		fl.emit(&ir.MapIterNext{Ok: ok, Key: k, Val: val, Iter: iter})
		fl.term(&ir.If{Cond: ok, Then: body, Else: done})
		fl.b = body
		for i, x := range []*ir.Local{k, val} {
			s := fl.temp(res[i].Type)
			fl.emit(&ir.Append{Dst: s, S: res[i], Elems: []ir.Value{fl.conv(g.copyOf(fl, x), anyT, nil)}})
			fl.emit(&ir.Store{Place: localPlace(res[i]), V: s})
		}
		fl.jump(head)
		fl.b = done
		fl.term(&ir.Return{})
	}
	fl.finish()
	return &ir.FuncRef{Func: f, Type: sig}
}

func (g *jsonGen) copyOf(fl *fnLowerer, v *ir.Local) ir.Value {
	pt := fl.ts().PointerTo(v.Type)
	p := fl.temp(pt)
	fl.emit(&ir.New{Dst: p})
	fl.emit(&ir.Store{Place: &ir.Place{Root: ir.DerefRoot{Ptr: p}, Type: v.Type}, V: v})
	return p
}

// canonical converts a basic or byte-slice value to the representation
// std/encoding/json reads: bool, int64, uint64, float64, string or []byte.
func (g *jsonGen) canonical(fl *fnLowerer, v ir.Value, t types.Type) ir.Value {
	var to types.Type
	switch jsontype.KindOf(t) {
	case jsontype.Bool:
		to = types.Typ[types.Bool]
	case jsontype.Int:
		to = types.Typ[types.Int64]
	case jsontype.Uint:
		to = types.Typ[types.Uint64]
	case jsontype.Float32, jsontype.Float64:
		to = types.Typ[types.Float64]
	case jsontype.String:
		to = types.Typ[types.String]
	case jsontype.Bytes:
		to = types.NewSlice(types.Typ[types.Uint8])
	}
	tt := fl.typ(to)
	if v.IRType() == tt {
		return v
	}
	fu, tu := v.IRType().U(), tt.U()
	kind := ir.ConvNop
	switch {
	case fu.Kind == ir.KFloat:
		kind = ir.ConvFloat
	case fu.Kind == ir.KInt && tu.Kind == ir.KInt && fu.Int != tu.Int:
		kind = ir.ConvInt
	}
	dst := fl.temp(tt)
	fl.emit(&ir.Convert{Dst: dst, X: v, Kind: kind})
	return dst
}

// fieldAccessor returns func(p any) any yielding a pointer to the field at
// fd.Index within *p, or nil when an embedded pointer on the path is nil.
func (g *jsonGen) fieldAccessor(t types.Type, fd jsontype.Field, sig *ir.Type) ir.Value {
	fl, f, ptr := g.newAccessor(t, "field_"+fd.Name, sig)
	ts := g.l.ts
	place := &ir.Place{Root: ir.DerefRoot{Ptr: ptr}, Type: ts.Of(t)}
	st := t
	for i, idx := range fd.Index {
		if i > 0 {
			if p, ok := types.Unalias(st).(*types.Pointer); ok {
				pv := fl.load(place, nil)
				c := fl.temp(ts.Bool())
				fl.emit(&ir.BinOp{Dst: c, Op: ir.Eq, X: pv, Y: &ir.Const{Type: pv.Type, Nil: true}})
				yes, no := f.NewBlock("nil"), f.NewBlock("embedded")
				fl.term(&ir.If{Cond: c, Then: yes, Else: no})
				fl.b = yes
				fl.term(&ir.Return{})
				fl.b = no
				st = p.Elem()
				place = &ir.Place{Root: ir.DerefRoot{Ptr: pv}, Type: ts.Of(st)}
			}
		}
		field := st.Underlying().(*types.Struct).Field(idx)
		place = withProj(place, ir.Proj{Field: idx, Type: ts.Of(field.Type())})
		st = field.Type()
	}
	addr := fl.temp(ts.PointerTo(place.Type))
	fl.emit(&ir.AddrOf{Dst: addr, Place: place})
	fl.emit(&ir.Store{Place: localPlace(f.Results[0]), V: fl.conv(addr, ts.Any(), nil)})
	fl.term(&ir.Return{})
	fl.finish()
	return &ir.FuncRef{Func: f, Type: sig}
}

func (fl *fnLowerer) zeroValue(t *ir.Type) ir.Value {
	switch t.U().Kind {
	case ir.KBool:
		return &ir.Const{Type: t, Val: constant.MakeBool(false)}
	case ir.KInt, ir.KFloat:
		return &ir.Const{Type: t, Val: constant.MakeInt64(0)}
	case ir.KString:
		return &ir.Const{Type: t, Val: constant.MakeString("")}
	}
	return &ir.Const{Type: t, Nil: true}
}

// jsonNilPointers fills std/encoding/json.isNilPointer once every function is
// lowered: it reports whether an interface holds a nil pointer of a type,
// converted to an interface somewhere in the program, that implements
// Marshaler or encoding.TextMarshaler, which encoding/json encodes as null.
func (l *Lowerer) jsonNilPointers() {
	var fn *ir.Func
	for obj, f := range l.funcs {
		if obj.Name() == "isNilPointer" && obj.Pkg() != nil && obj.Pkg().Path() == catalog.JSONPackage && obj.Signature().Recv() == nil {
			fn = f
		}
	}
	if fn == nil {
		return
	}
	var ptrs []*ir.Type
	for _, t := range l.ts.All {
		if !t.Boxed || t.U().Kind != ir.KPointer {
			continue
		}
		if jsontype.Flags(t.Go)&(jsontype.MarshalJSON|jsontype.MarshalText) != 0 {
			ptrs = append(ptrs, t)
		}
	}
	fn.Blocks = nil
	fl := l.newFn(nil, fn)
	fl.start()
	result := fn.Results[0]
	fl.emit(&ir.DeclVar{L: result})
	for _, t := range ptrs {
		v := fl.temp(t)
		ok := fl.temp(l.ts.Bool())
		fl.emit(&ir.TypeAssert{Dst: v, Ok: ok, X: fn.Params[0], T: t})
		match, next := fn.NewBlock("match"), fn.NewBlock("next")
		fl.term(&ir.If{Cond: ok, Then: match, Else: next})
		fl.b = match
		isNil := fl.temp(l.ts.Bool())
		fl.emit(&ir.BinOp{Dst: isNil, Op: ir.Eq, X: v, Y: &ir.Const{Type: t, Nil: true}})
		fl.emit(&ir.Store{Place: localPlace(result), V: isNil})
		fl.term(&ir.Return{})
		fl.b = next
	}
	fl.term(&ir.Return{})
	fl.finish()
}

package ir

// ComputeEffects determines which functions may suspend. Channel
// operations, select, and suspending capability calls seed the analysis.
// Calls through function values and interfaces are resolved conservatively
// to every function in the program that could be the callee: functions
// whose value is taken with a matching signature, and methods of boxed
// types with the called method identity. Effects propagate through the call
// graph to a fixed point. The result marks Func.MaySuspend and
// Call.Suspends, and sets Program.Cooperative.
func ComputeEffects(p *Program) {
	valueFuncs := map[*Type][]*Func{} // by value signature
	methodImpls := map[string][]*Func{}
	sigOf := func(f *Func, dropRecv bool) *Type {
		if !dropRecv {
			return f.Sig
		}
		return nil
	}
	_ = sigOf
	for _, f := range p.Funcs {
		for _, b := range f.Blocks {
			for _, in := range b.Instrs {
				switch i := in.(type) {
				case *MakeClosure:
					valueFuncs[i.Dst.Type.U()] = append(valueFuncs[i.Dst.Type.U()], i.Func)
				case *MakeBound:
					valueFuncs[i.Dst.Type.U()] = append(valueFuncs[i.Dst.Type.U()], i.Func)
				}
				forEachValue(in, func(v Value) {
					if fr, ok := v.(*FuncRef); ok {
						valueFuncs[fr.Type.U()] = append(valueFuncs[fr.Type.U()], fr.Func)
					}
				})
			}
		}
	}
	for _, t := range p.Types.All {
		for _, m := range t.MethodSet {
			methodImpls[m.ID] = append(methodImpls[m.ID], m.Func)
		}
	}
	candidates := func(c *Call) []*Func {
		switch c.Kind {
		case CallStatic:
			return []*Func{c.Func}
		case CallValue:
			return valueFuncs[c.Fn.IRType().U()]
		case CallInterface:
			return methodImpls[c.Method]
		}
		return nil
	}
	// Bound interface method values may call any implementation.
	for _, f := range p.Funcs {
		for _, b := range f.Blocks {
			for _, in := range b.Instrs {
				if ib, ok := in.(*MakeIfaceBound); ok {
					valueFuncs[ib.Dst.Type.U()] = append(valueFuncs[ib.Dst.Type.U()], methodImpls[ib.Method]...)
				}
			}
		}
	}
	direct := func(in Instr) bool {
		switch i := in.(type) {
		case *Send, *Recv, *Select:
			return true
		case *Call:
			return i.Kind == CallExtern && i.Extern.MaySuspend
		case *Defer:
			return i.Call.Kind == CallExtern && i.Call.Extern.MaySuspend
		}
		return false
	}
	for _, f := range p.Funcs {
		for _, b := range f.Blocks {
			for _, in := range b.Instrs {
				switch in.(type) {
				case *Send, *Recv, *Select, *Go, *MakeChan, *Close:
					p.Cooperative = true
				}
				if c, ok := in.(*Call); ok && c.Kind == CallExtern && c.Extern.MaySuspend {
					p.Cooperative = true
				}
				if direct(in) {
					f.MaySuspend = true
				}
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, f := range p.Funcs {
			if f.MaySuspend {
				continue
			}
		scan:
			for _, b := range f.Blocks {
				for _, in := range b.Instrs {
					var c *Call
					switch i := in.(type) {
					case *Call:
						c = i
					case *Defer:
						c = i.Call
					}
					if c == nil {
						continue
					}
					for _, cand := range candidates(c) {
						if cand.MaySuspend {
							f.MaySuspend = true
							changed = true
							break scan
						}
					}
				}
			}
		}
	}
	for _, f := range p.Funcs {
		for _, b := range f.Blocks {
			for _, in := range b.Instrs {
				var c *Call
				switch i := in.(type) {
				case *Call:
					c = i
				case *Defer:
					c = i.Call
				case *Go:
					c = i.Call
				}
				if c == nil {
					continue
				}
				if c.Kind == CallExtern {
					c.Suspends = c.Extern.MaySuspend
					continue
				}
				for _, cand := range candidates(c) {
					if cand.MaySuspend {
						c.Suspends = true
						break
					}
				}
			}
		}
	}
}

// forEachValue calls fn for each operand of an instruction.
func forEachValue(in Instr, fn func(Value)) {
	vals := func(vs ...Value) {
		for _, v := range vs {
			if v != nil {
				fn(v)
			}
		}
	}
	place := func(p *Place) {
		switch r := p.Root.(type) {
		case DerefRoot:
			vals(r.Ptr)
		case SliceRoot:
			vals(r.Slice, r.Index)
		case ValueRoot:
			vals(r.Value)
		}
		for _, pr := range p.Path {
			if pr.Index != nil {
				vals(pr.Index)
			}
		}
	}
	switch i := in.(type) {
	case *DeclVar:
		if i.Init != nil {
			vals(i.Init)
		}
	case *Assign:
		vals(i.Src)
	case *Load:
		place(i.Place)
	case *Store:
		place(i.Place)
		vals(i.V)
	case *AddrOf:
		place(i.Place)
	case *UnOp:
		vals(i.X)
	case *BinOp:
		vals(i.X, i.Y)
	case *Convert:
		vals(i.X)
	case *MakeInterface:
		vals(i.X)
	case *TypeAssert:
		vals(i.X)
	case *Call:
		vals(i.Fn, i.Recv)
		vals(i.Args...)
	case *Defer:
		vals(i.Call.Fn, i.Call.Recv)
		vals(i.Call.Args...)
	case *Go:
		vals(i.Call.Fn, i.Call.Recv)
		vals(i.Call.Args...)
	case *MakeBound:
		vals(i.Recv)
	case *MakeIfaceBound:
		vals(i.Recv)
	case *Len:
		vals(i.X)
	case *Cap:
		vals(i.X)
	case *MakeSlice:
		vals(i.Len, i.Cap)
	case *Append:
		vals(i.S, i.Spread)
		vals(i.Elems...)
	case *Copy:
		vals(i.Dst, i.Src)
	case *IndexString:
		vals(i.S, i.I)
	case *SliceOp:
		vals(i.X, i.Lo, i.Hi, i.Max)
	case *MapLookup:
		vals(i.M, i.K)
	case *MapStore:
		vals(i.M, i.K, i.V)
	case *MapDelete:
		vals(i.M, i.K)
	case *Clear:
		vals(i.X)
	case *DecodeRune:
		vals(i.S, i.I)
	case *MapIterInit:
		vals(i.M)
	case *Print:
		vals(i.Args...)
	case *MakeChan:
		vals(i.Size)
	case *Send:
		vals(i.Ch, i.V)
	case *Recv:
		vals(i.Ch)
	case *Close:
		vals(i.Ch)
	case *Select:
		for _, c := range i.Cases {
			vals(c.Ch, c.V)
		}
	}
}

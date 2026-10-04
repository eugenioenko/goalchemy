package ir

import (
	"fmt"
	"go/constant"
	"strings"
)

// Verify checks structural invariants of a function: every block has a
// terminator, branch targets belong to the function, locals belong to the
// function, and operand types agree where the IR requires it.
func Verify(f *Func) error {
	if len(f.Blocks) == 0 {
		return fmt.Errorf("%s: function has no blocks", f.Name)
	}
	blocks := map[*Block]bool{}
	for _, b := range f.Blocks {
		blocks[b] = true
	}
	locals := map[*Local]bool{}
	for _, l := range f.Locals {
		locals[l] = true
	}
	var errs []string
	bad := func(b *Block, format string, args ...any) {
		errs = append(errs, fmt.Sprintf("%s b%d: %s", f.Name, b.ID, fmt.Sprintf(format, args...)))
	}
	checkVal := func(b *Block, v Value) {
		switch v := v.(type) {
		case nil:
			bad(b, "nil operand")
		case *Local:
			if !locals[v] {
				bad(b, "local %s#%d does not belong to the function", v.Name, v.ID)
			}
		case *Const:
			if v.Type == nil {
				bad(b, "untyped constant")
			} else if !v.Nil && v.Val == nil {
				bad(b, "constant without value")
			} else if v.Val != nil && v.Val.Kind() == constant.Int && v.Type.U().Kind != KInt && v.Type.U().Kind != KFloat {
				bad(b, "integer constant of type %s", v.Type)
			}
		case *FuncRef:
			if v.Func == nil {
				bad(b, "nil function reference")
			}
		}
	}
	checkLocal := func(b *Block, l *Local) {
		if l != nil && !locals[l] {
			bad(b, "local %s#%d does not belong to the function", l.Name, l.ID)
		}
	}
	checkPlace := func(b *Block, p *Place) {
		switch r := p.Root.(type) {
		case LocalRoot:
			checkLocal(b, r.Local)
		case DerefRoot:
			checkVal(b, r.Ptr)
			if r.Ptr != nil && r.Ptr.IRType().U().Kind != KPointer {
				bad(b, "dereference of non-pointer %s", r.Ptr.IRType())
			}
		case SliceRoot:
			checkVal(b, r.Slice)
			checkVal(b, r.Index)
		case ValueRoot:
			checkVal(b, r.Value)
		case GlobalRoot:
		default:
			bad(b, "place without root")
		}
		for _, pr := range p.Path {
			if pr.Index != nil {
				checkVal(b, pr.Index)
			}
		}
	}
	sameType := func(b *Block, what string, want *Type, v Value) {
		if v == nil || want == nil {
			return
		}
		if got := v.IRType(); got != want {
			bad(b, "%s: have %s, want %s", what, got, want)
		}
	}
	for _, b := range f.Blocks {
		for _, in := range b.Instrs {
			switch i := in.(type) {
			case *DeclVar:
				checkLocal(b, i.L)
				if i.Init != nil {
					checkVal(b, i.Init)
					sameType(b, "declvar", i.L.Type, i.Init)
				}
			case *Assign:
				checkLocal(b, i.Dst)
				checkVal(b, i.Src)
				sameType(b, "assign", i.Dst.Type, i.Src)
			case *Load:
				checkLocal(b, i.Dst)
				checkPlace(b, i.Place)
				if i.Dst.Type != i.Place.Type {
					bad(b, "load type %s from place of %s", i.Dst.Type, i.Place.Type)
				}
			case *Store:
				checkPlace(b, i.Place)
				checkVal(b, i.V)
				sameType(b, "store", i.Place.Type, i.V)
				if _, ok := i.Place.Root.(ValueRoot); ok {
					bad(b, "store to a non-addressable value")
				}
			case *AddrOf:
				checkPlace(b, i.Place)
				if i.Place.HasElement() {
					bad(b, "address of an element")
				}
			case *BinOp:
				checkVal(b, i.X)
				checkVal(b, i.Y)
				if i.Op != Shl && i.Op != Shr {
					sameType(b, "binop "+i.Op.String(), i.X.IRType(), i.Y)
				}
			case *UnOp:
				checkVal(b, i.X)
			case *Call:
				for _, a := range i.Args {
					checkVal(b, a)
				}
				for _, d := range i.Dsts {
					checkLocal(b, d)
				}
				if i.Kind == CallStatic && i.Func != nil {
					if len(i.Args) != len(i.Func.Sig.Params) {
						bad(b, "call of %s with %d args, want %d", i.Func.Name, len(i.Args), len(i.Func.Sig.Params))
					} else {
						for j, a := range i.Args {
							sameType(b, "argument to "+i.Func.Name, i.Func.Sig.Params[j], a)
						}
					}
				}
			case *Defer:
				for _, a := range i.Call.Args {
					checkVal(b, a)
				}
			}
		}
		switch t := b.Term.(type) {
		case nil:
			bad(b, "missing terminator")
		case *Jump:
			if !blocks[t.Target] {
				bad(b, "jump to foreign block")
			}
		case *If:
			checkVal(b, t.Cond)
			if !blocks[t.Then] || !blocks[t.Else] {
				bad(b, "branch to foreign block")
			}
			if t.Cond != nil && t.Cond.IRType().U().Kind != KBool {
				bad(b, "branch on non-Boolean %s", t.Cond.IRType())
			}
		case *Panic:
			checkVal(b, t.X)
		case *Pause:
			if !blocks[t.Next] || t.Next.ResumeOf != t.Op {
				bad(b, "pause continuation is not its resume block")
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// Successors returns the blocks a terminator may transfer control to.
func Successors(t Terminator) []*Block {
	switch t := t.(type) {
	case *Jump:
		return []*Block{t.Target}
	case *If:
		return []*Block{t.Then, t.Else}
	case *Pause:
		return []*Block{t.Next}
	}
	return nil
}

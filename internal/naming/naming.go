// Package naming assigns deterministic private identifiers to emitted IR.
// It never changes source identities, public APIs, or the shared IR.
package naming

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/ir"
)

// Identifier encodes a source fragment using ASCII identifier characters.
// Callers add a category prefix and an identity suffix, so reserved words,
// source punctuation, and names from different scopes cannot collide.
func Identifier(source string) string {
	var b strings.Builder
	for _, r := range source {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		case r == '.', r == '/', r == '-':
			b.WriteByte('_')
		default:
			fmt.Fprintf(&b, "_u%x_", r)
		}
	}
	return b.String()
}

func compact(p *ir.Program) bool { return p != nil && p.CompactNames }

// Names is an immutable naming index for one emission. Constructing it once
// avoids rescanning the program at each function/global reference.
type Names struct {
	p       *ir.Program
	symbols map[string]string
	types   map[*ir.Type]*ir.Type
}

// New indexes private symbols and canonical representatives without modifying p.
func New(p *ir.Program) *Names {
	n := &Names{p: p, symbols: map[string]string{}, types: map[*ir.Type]*ir.Type{}}
	if p == nil {
		return n
	}
	for _, f := range p.Funcs {
		n.symbols[f.Sym] = function(p, f)
	}
	for _, g := range p.Globals {
		n.symbols[g.Sym] = global(p, g)
	}
	if p.Types != nil {
		for _, t := range p.Types.All {
			if t.Kind == ir.KNamed {
				u := t.U()
				old := n.types[u]
				if old == nil || t.Name+"\x00"+t.Pkg < old.Name+"\x00"+old.Pkg {
					n.types[u] = t
				}
			}
		}
	}
	return n
}

func (n *Names) Type(t *ir.Type, prefix string) string {
	representative := t
	if t.Kind != ir.KNamed && n.types[t] != nil {
		representative = n.types[t]
	}
	return typeName(n.p, t, representative, prefix)
}

func (n *Names) Local(l *ir.Local, separator string) string { return Local(n.p, l, separator) }
func (n *Names) Field(t *ir.Type, index int, prefix string) string {
	return Field(n.p, t, index, prefix)
}
func (n *Names) Symbol(oldSym string) string {
	if s, ok := n.symbols[oldSym]; ok {
		return s
	}
	return oldSym
}

// Type names a target representation of t. Pass the target's category prefix,
// such as T_, S$, or S. For shared underlying representations, the smallest
// source-qualified named type is the deterministic readable representative.
func Type(p *ir.Program, t *ir.Type, prefix string) string {
	n := t
	if t.Kind != ir.KNamed && p != nil && p.Types != nil {
		for _, candidate := range p.Types.All {
			if candidate.Kind == ir.KNamed && candidate.U() == t &&
				(n == t || candidate.Name+"\x00"+candidate.Pkg < n.Name+"\x00"+n.Pkg) {
				n = candidate
			}
		}
	}
	return typeName(p, t, n, prefix)
}

func typeName(p *ir.Program, t, representative *ir.Type, prefix string) string {
	id := strconv.Itoa(t.ID)
	if compact(p) {
		return prefix + id
	}
	return prefix + Identifier(typeDescription(representative, 0)) + "_" + id
}

func typeDescription(t *ir.Type, depth int) string {
	if t.Name != "" {
		return t.Name
	}
	if t.Basic != "" {
		return t.Basic
	}
	if depth >= 3 {
		return t.Kind.String()
	}
	switch t.Kind {
	case ir.KArray:
		return "array_" + strconv.FormatInt(t.Len, 10) + "_" + typeDescription(t.Elem, depth+1)
	case ir.KSlice, ir.KPointer, ir.KChan, ir.KMapIter:
		return t.Kind.String() + "_" + typeDescription(t.Elem, depth+1)
	case ir.KMap:
		return "map_" + typeDescription(t.Key, depth+1) + "_" + typeDescription(t.Elem, depth+1)
	default:
		return t.Kind.String()
	}
}

// Local names a function-scoped cell. separator must be legal in the target
// language, normally "_" or "$". IDs distinguish shadowed source locals.
func Local(p *ir.Program, l *ir.Local, separator string) string {
	id := strconv.Itoa(l.ID)
	if compact(p) {
		return "v" + id
	}
	name := l.Name
	if name == "" || name == "_" {
		switch l.Kind {
		case ir.LParam:
			name = "param"
		case ir.LResult:
			name = "result"
		case ir.LEnv:
			name = "capture"
		case ir.LVar:
			name = "var"
		default:
			name = "temp"
		}
	}
	return "v_" + Identifier(name) + separator + id
}

// Symbol maps a lowered function or global symbol to its private target name.
// Capability/runtime symbols and other unknown symbols pass through unchanged.
// Public wrappers must keep their established API names independently of this.
func Symbol(p *ir.Program, oldSym string) string {
	if p == nil {
		return oldSym
	}
	for _, f := range p.Funcs {
		if f.Sym != oldSym {
			continue
		}
		return function(p, f)
	}
	for _, g := range p.Globals {
		if g.Sym != oldSym {
			continue
		}
		return global(p, g)
	}
	return oldSym
}

func function(p *ir.Program, f *ir.Func) string {
	id := strconv.Itoa(f.ID)
	if compact(p) {
		return "f" + id
	}
	name := f.Name
	if name == "" {
		name = f.Sym
	}
	if name == "$init" {
		name = "init"
	}
	if name == "$entry" {
		name = "entry"
	}
	return "fn_" + Identifier(name) + "_" + id
}

func global(p *ir.Program, g *ir.Global) string {
	id := strconv.Itoa(g.ID)
	if compact(p) {
		return "g" + id
	}
	name := g.Sym
	if g.Name != "" {
		name = g.Name
		if g.Pkg != "" {
			name = path.Base(g.Pkg) + "_" + name
		}
	}
	return "global_" + Identifier(name) + "_" + id
}

// Field names private aggregate storage. Public fields and Go embedded fields
// retain their API/embedding rules and should not use this helper directly.
func Field(p *ir.Program, t *ir.Type, index int, prefix string) string {
	id := strconv.Itoa(index)
	if compact(p) {
		return prefix + id
	}
	return prefix + Identifier(t.U().Fields[index].Name) + "_" + id
}

// Package ir defines Goalchemy's target-independent typed intermediate
// representation: typed functions made of basic blocks with explicit branch
// targets, local storage cells, and three-address instructions whose
// evaluation order and copy boundaries are already decided by lowering.
package ir

import (
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/types/typeutil"
)

type Kind uint8

const (
	KBool Kind = iota + 1
	KInt
	KString
	KStruct
	KArray
	KSlice
	KMap
	KPointer
	KFunc
	KInterface
	KNamed
	KTuple
	KChan
	// KMapIter is the type of map iterator temporaries; Elem is the map type.
	KMapIter
	// KOpaque is an external capability type represented by a runtime
	// handle, such as sync.Mutex or context.Context.
	KOpaque
	// KFloat is an IEEE binary32 or binary64 scalar; FloatBits records width.
	KFloat
)

func (k Kind) String() string {
	return [...]string{"invalid", "bool", "int", "string", "struct", "array", "slice", "map", "pointer", "func", "interface", "named", "tuple", "chan", "mapiter", "opaque", "float"}[k]
}

// IntKind identifies the representation of an integer type.
type IntKind uint8

const (
	I8 IntKind = iota + 1
	I16
	I32
	I64
	U8
	U16
	U32
	U64
)

func (k IntKind) Bits() int {
	switch k {
	case I8, U8:
		return 8
	case I16, U16:
		return 16
	case I32, U32:
		return 32
	}
	return 64
}

func (k IntKind) Signed() bool { return k <= I64 }

func (k IntKind) String() string {
	return [...]string{"?", "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64"}[k]
}

type Field struct {
	Name     string
	Type     *Type
	Embedded bool
	// ID is the Go field identity (package-qualified when unexported).
	ID string
}

type Method struct {
	// ID is the Go method identity: the name if exported, else pkgpath.name.
	ID   string
	Name string
	Sig  *Type
}

// MethodEntry binds a method identity in a type's method set to the function
// that implements it with this type as the receiver.
type MethodEntry struct {
	ID   string
	Name string
	Func *Func
}

// Type is a canonical IR type. Identical Go types map to one *Type.
type Type struct {
	ID   int
	Kind Kind

	// Basic name for bool, string, and integers: "int", "uint8", ...
	Basic     string
	Int       IntKind
	FloatBits int

	Elem *Type
	Key  *Type
	Len  int64

	Fields []Field

	Params   []*Type
	Results  []*Type
	Variadic bool

	// Interface methods sorted by ID.
	Methods []Method

	// Named types.
	Name       string // source-qualified name such as main.T
	Pkg        string
	Obj        string
	Underlying *Type

	// MethodSet lists the methods of this type, sorted by ID, once lowering
	// has computed it (for every type that is converted to an interface).
	MethodSet     []*MethodEntry
	MethodSetDone bool
	// Boxed is set when values of this type are stored in interfaces.
	Boxed bool

	// Elements of tuples.
	Elems []*Type

	ChanDir types.ChanDir

	// OpaqueRef marks opaque types with reference semantics (interfaces such
	// as context.Context); other opaque types are values like sync.Mutex.
	OpaqueRef bool

	Go types.Type
}

// U returns the underlying type.
func (t *Type) U() *Type {
	if t.Kind == KNamed {
		return t.Underlying
	}
	return t
}

func (t *Type) IsInterface() bool { return t.U().Kind == KInterface }

// IsAggregate reports whether values of t have value semantics requiring
// explicit copies on hosts that share objects.
func (t *Type) IsAggregate() bool {
	u := t.U()
	return u.Kind == KStruct || u.Kind == KArray || u.Kind == KOpaque && !u.OpaqueRef
}

// HasAggregate reports whether copying t must copy nested storage.
func (t *Type) Comparable() bool {
	return types.Comparable(t.Go)
}

func (t *Type) String() string {
	if t.Kind == KNamed {
		return t.Name
	}
	return TypeString(t)
}

// SourceName renders a type the way the Go runtime prints it, using
// source-qualified names for named types.
func TypeString(t *Type) string {
	switch t.Kind {
	case KBool, KInt, KString, KFloat:
		return t.Basic
	case KNamed, KOpaque:
		return t.Name
	case KSlice:
		return "[]" + TypeString(t.Elem)
	case KArray:
		return fmt.Sprintf("[%d]%s", t.Len, TypeString(t.Elem))
	case KPointer:
		return "*" + TypeString(t.Elem)
	case KMap:
		return "map[" + TypeString(t.Key) + "]" + TypeString(t.Elem)
	case KChan:
		switch t.ChanDir {
		case types.SendOnly:
			return "chan<- " + TypeString(t.Elem)
		case types.RecvOnly:
			return "<-chan " + TypeString(t.Elem)
		}
		return "chan " + TypeString(t.Elem)
	case KStruct:
		var b strings.Builder
		b.WriteString("struct {")
		for i, f := range t.Fields {
			if i > 0 {
				b.WriteString(";")
			}
			b.WriteString(" ")
			if !f.Embedded {
				b.WriteString(f.Name + " ")
			}
			b.WriteString(TypeString(f.Type))
		}
		if len(t.Fields) > 0 {
			b.WriteString(" ")
		}
		b.WriteString("}")
		return b.String()
	case KInterface:
		if len(t.Methods) == 0 {
			return "interface {}"
		}
		var b strings.Builder
		b.WriteString("interface {")
		for i, m := range t.Methods {
			if i > 0 {
				b.WriteString(";")
			}
			b.WriteString(" " + m.Name + strings.TrimPrefix(TypeString(m.Sig), "func"))
		}
		b.WriteString(" }")
		return b.String()
	case KFunc:
		var b strings.Builder
		b.WriteString("func(")
		for i, p := range t.Params {
			if i > 0 {
				b.WriteString(", ")
			}
			if t.Variadic && i == len(t.Params)-1 {
				b.WriteString("..." + TypeString(p.Elem))
			} else {
				b.WriteString(TypeString(p))
			}
		}
		b.WriteString(")")
		switch len(t.Results) {
		case 0:
		case 1:
			b.WriteString(" " + TypeString(t.Results[0]))
		default:
			b.WriteString(" (")
			for i, r := range t.Results {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(TypeString(r))
			}
			b.WriteString(")")
		}
		return b.String()
	case KTuple:
		parts := make([]string, len(t.Elems))
		for i, e := range t.Elems {
			parts[i] = TypeString(e)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}
	return "?"
}

// Types interns IR types by Go type identity.
type Types struct {
	All   []*Type
	cache typeutil.Map
	iters map[*Type]*Type
	// Opaque reports external named types represented by runtime handles.
	Opaque func(*types.TypeName) bool
}

// MapIter returns the iterator type for map type m.
func (ts *Types) MapIter(m *Type) *Type {
	if ts.iters == nil {
		ts.iters = map[*Type]*Type{}
	}
	if t, ok := ts.iters[m]; ok {
		return t
	}
	t := ts.add(&Type{Kind: KMapIter, Elem: m})
	ts.iters[m] = t
	return t
}

func NewTypes() *Types { return &Types{} }

func (ts *Types) add(t *Type) *Type {
	t.ID = len(ts.All)
	ts.All = append(ts.All, t)
	return t
}

var intKinds = map[types.BasicKind]IntKind{
	types.Int8: I8, types.Int16: I16, types.Int32: I32, types.Int64: I64, types.Int: I64,
	types.Uint8: U8, types.Uint16: U16, types.Uint32: U32, types.Uint64: U64, types.Uint: U64,
}

// Of returns the IR type for a Go type. Untyped constants are defaulted.
func (ts *Types) Of(gt types.Type) *Type {
	gt = types.Unalias(gt)
	if b, ok := gt.(*types.Basic); ok && b.Info()&types.IsUntyped != 0 {
		gt = types.Default(gt)
		if b2, ok := gt.(*types.Basic); ok && b2.Kind() == types.UntypedNil {
			panic("ir: untyped nil has no IR type")
		}
	}
	if t, ok := ts.cache.At(gt).(*Type); ok {
		return t
	}
	t := &Type{Go: gt}
	ts.cache.Set(gt, t)
	switch g := gt.(type) {
	case *types.Basic:
		switch {
		case g.Kind() == types.Bool:
			t.Kind, t.Basic = KBool, "bool"
		case g.Kind() == types.String:
			t.Kind, t.Basic = KString, "string"
		case g.Kind() == types.Float32 || g.Kind() == types.Float64:
			t.Kind, t.Basic, t.FloatBits = KFloat, g.Name(), 64
			if g.Kind() == types.Float32 {
				t.FloatBits = 32
			}
		default:
			k, ok := intKinds[g.Kind()]
			if !ok {
				panic(fmt.Sprintf("ir: unsupported basic type %s", g))
			}
			t.Kind, t.Int, t.Basic = KInt, k, types.Typ[g.Kind()].Name()
		}
	case *types.Named:
		obj := g.Obj()
		if ts.Opaque != nil && ts.Opaque(obj) {
			t.Kind, t.Obj = KOpaque, obj.Name()
			t.Pkg = obj.Pkg().Path()
			t.Name = obj.Pkg().Name() + "." + obj.Name()
			_, t.OpaqueRef = g.Underlying().(*types.Interface)
			return ts.add(t)
		}
		t.Kind = KNamed
		t.Obj = obj.Name()
		if obj.Pkg() != nil {
			t.Pkg = obj.Pkg().Path()
			t.Name = obj.Pkg().Name() + "." + obj.Name()
		} else {
			t.Name = obj.Name()
		}
		ts.add(t)
		t.Underlying = ts.Of(g.Underlying())
		return t
	case *types.Pointer:
		t.Kind = KPointer
		ts.add(t)
		t.Elem = ts.Of(g.Elem())
		return t
	case *types.Slice:
		t.Kind = KSlice
		ts.add(t)
		t.Elem = ts.Of(g.Elem())
		return t
	case *types.Array:
		t.Kind, t.Len = KArray, g.Len()
		ts.add(t)
		t.Elem = ts.Of(g.Elem())
		return t
	case *types.Map:
		t.Kind = KMap
		ts.add(t)
		t.Key = ts.Of(g.Key())
		t.Elem = ts.Of(g.Elem())
		return t
	case *types.Chan:
		t.Kind, t.ChanDir = KChan, g.Dir()
		ts.add(t)
		t.Elem = ts.Of(g.Elem())
		return t
	case *types.Struct:
		t.Kind = KStruct
		ts.add(t)
		for i := 0; i < g.NumFields(); i++ {
			f := g.Field(i)
			t.Fields = append(t.Fields, Field{Name: f.Name(), Type: ts.Of(f.Type()), Embedded: f.Embedded(), ID: f.Id()})
		}
		return t
	case *types.Signature:
		t.Kind, t.Variadic = KFunc, g.Variadic()
		ts.add(t)
		for i := 0; i < g.Params().Len(); i++ {
			t.Params = append(t.Params, ts.Of(g.Params().At(i).Type()))
		}
		for i := 0; i < g.Results().Len(); i++ {
			t.Results = append(t.Results, ts.Of(g.Results().At(i).Type()))
		}
		return t
	case *types.Interface:
		t.Kind = KInterface
		ts.add(t)
		for i := 0; i < g.NumMethods(); i++ {
			m := g.Method(i)
			t.Methods = append(t.Methods, Method{ID: MethodID(m), Name: m.Name(), Sig: ts.Of(m.Type())})
		}
		return t
	case *types.Tuple:
		t.Kind = KTuple
		ts.add(t)
		for i := 0; i < g.Len(); i++ {
			t.Elems = append(t.Elems, ts.Of(g.At(i).Type()))
		}
		return t
	default:
		panic(fmt.Sprintf("ir: unsupported type %T %s", gt, gt))
	}
	return ts.add(t)
}

// Recv-free signature type for a method (receiver dropped).
func (ts *Types) Sig(sig *types.Signature) *Type {
	return ts.Of(types.NewSignatureType(nil, nil, nil, sig.Params(), sig.Results(), sig.Variadic()))
}

func (ts *Types) Basic(k types.BasicKind) *Type { return ts.Of(types.Typ[k]) }
func (ts *Types) Bool() *Type                   { return ts.Basic(types.Bool) }
func (ts *Types) IntT() *Type                   { return ts.Basic(types.Int) }
func (ts *Types) String() *Type                 { return ts.Basic(types.String) }
func (ts *Types) Any() *Type                    { return ts.Of(types.NewInterfaceType(nil, nil)) }
func (ts *Types) Error() *Type                  { return ts.Of(types.Universe.Lookup("error").Type()) }

func (ts *Types) SliceOf(e *Type) *Type   { return ts.Of(types.NewSlice(e.Go)) }
func (ts *Types) PointerTo(e *Type) *Type { return ts.Of(types.NewPointer(e.Go)) }

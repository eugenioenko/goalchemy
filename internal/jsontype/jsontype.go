// Package jsontype describes Go types for std/encoding/json the way
// encoding/json's reflection does, so the language gate and the lowering of
// json.Marshal agree on encoders, struct fields and unsupported types.
package jsontype

import (
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Package is the import path of std/encoding/json.
const Package = "github.com/eugenioenko/goalchemy/std/encoding/json"

// Kinds match the kind constants in std/encoding/json.
const (
	Bool = iota + 1
	Int
	Uint
	Float32
	Float64
	String
	Bytes
	Struct
	Slice
	Array
	Map
	Pointer
	Interface
)

// Flags match the encoder flags in std/encoding/json.
const (
	MarshalJSON = 1 << iota
	AddrMarshalJSON
	MarshalText
	AddrMarshalText
)

var (
	byteSlice   = types.NewSlice(types.Typ[types.Uint8])
	errorType   = types.Universe.Lookup("error").Type()
	marshaler   = method("MarshalJSON", nil, byteSlice, errorType)
	textMarshal = method("MarshalText", nil, byteSlice, errorType)
)

func method(name string, params []types.Type, results ...types.Type) *types.Interface {
	var ps, rs []*types.Var
	for _, p := range params {
		ps = append(ps, types.NewVar(token.NoPos, nil, "", p))
	}
	for _, r := range results {
		rs = append(rs, types.NewVar(token.NoPos, nil, "", r))
	}
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(ps...), types.NewTuple(rs...), false)
	return types.NewInterfaceType([]*types.Func{types.NewFunc(token.NoPos, nil, name, sig)}, nil).Complete()
}

// Flags reports which marshaler methods encoding/json would call for t.
func Flags(t types.Type) int {
	f := 0
	if types.Implements(t, marshaler) {
		f |= MarshalJSON
	}
	if types.Implements(t, textMarshal) {
		f |= MarshalText
	}
	if _, isPtr := t.Underlying().(*types.Pointer); !isPtr && !types.IsInterface(t) {
		p := types.NewPointer(t)
		if types.Implements(p, marshaler) {
			f |= AddrMarshalJSON
		}
		if types.Implements(p, textMarshal) {
			f |= AddrMarshalText
		}
	}
	return f
}

// Leaf reports whether encoding t never inspects its structure because t
// itself implements Marshaler or encoding.TextMarshaler.
func Leaf(t types.Type) bool { return Flags(t)&(MarshalJSON|MarshalText) != 0 }

// IsNumber reports whether t is json.Number.
func IsNumber(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Name() == "Number" && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == Package
}

// KindOf returns t's encoder kind, or 0 when encoding/json cannot encode it.
func KindOf(t types.Type) int {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Kind() == types.Bool:
			return Bool
		case u.Info()&types.IsInteger != 0 && u.Info()&types.IsUnsigned != 0:
			return Uint
		case u.Info()&types.IsInteger != 0:
			return Int
		case u.Kind() == types.Float32:
			return Float32
		case u.Kind() == types.Float64:
			return Float64
		case u.Kind() == types.String:
			return String
		}
	case *types.Slice:
		if b, ok := u.Elem().Underlying().(*types.Basic); ok && b.Kind() == types.Uint8 {
			p := types.NewPointer(u.Elem())
			if !types.Implements(p, marshaler) && !types.Implements(p, textMarshal) {
				return Bytes
			}
		}
		return Slice
	case *types.Array:
		return Array
	case *types.Map:
		return Map
	case *types.Pointer:
		return Pointer
	case *types.Interface:
		return Interface
	case *types.Struct:
		return Struct
	}
	return 0
}

// ValidMapKey reports whether encoding/json can encode keys of type k.
func ValidMapKey(k types.Type) bool {
	switch KindOf(k) {
	case String, Int, Uint:
		return true
	}
	return types.Implements(k, textMarshal)
}

// Field is a struct field as encoding/json's typeFields resolves it.
type Field struct {
	Name      string
	Key       string
	Index     []int
	Type      types.Type
	Var       *types.Var
	OmitEmpty bool
	OmitZero  bool
	Quoted    bool
	ViaPtr    bool
	tag       bool
}

type candidate struct {
	typ    types.Type
	index  []int
	viaPtr bool
}

// Fields returns the fields encoding/json encodes for struct type t, in
// encoding order, following embedded structs with Go's dominance rules.
func Fields(t types.Type) []Field {
	current := []candidate{}
	next := []candidate{{typ: t}}
	var count, nextCount map[string]int
	visited := map[string]bool{}
	var fields []Field
	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[string]int{}
		for _, f := range current {
			id := typeID(f.typ)
			if visited[id] {
				continue
			}
			visited[id] = true
			st := f.typ.Underlying().(*types.Struct)
			for i := 0; i < st.NumFields(); i++ {
				sf := st.Field(i)
				if sf.Embedded() {
					et := sf.Type()
					if p, ok := types.Unalias(et).(*types.Pointer); ok {
						et = p.Elem()
					}
					if !sf.Exported() {
						if _, ok := et.Underlying().(*types.Struct); !ok {
							continue
						}
					}
				} else if !sf.Exported() {
					continue
				}
				tag := reflectTagGet(st.Tag(i), "json")
				if tag == "-" {
					continue
				}
				name, opts := parseTag(tag)
				if !isValidTag(name) {
					name = ""
				}
				index := append(append([]int(nil), f.index...), i)
				ft := sf.Type()
				fp, ftPtr := types.Unalias(ft).(*types.Pointer)
				if ftPtr {
					ft = fp.Elem()
				}
				quoted := false
				if hasOpt(opts, "string") {
					switch KindOf(ft) {
					case Bool, Int, Uint, Float32, Float64, String:
						quoted = true
					}
				}
				_, isStruct := ft.Underlying().(*types.Struct)
				if name != "" || !sf.Embedded() || !isStruct {
					tagged := name != ""
					if name == "" {
						name = sf.Name()
					}
					field := Field{Name: name, tag: tagged, Index: index, Type: sf.Type(), Var: sf,
						OmitEmpty: hasOpt(opts, "omitempty"), OmitZero: hasOpt(opts, "omitzero"), Quoted: quoted, ViaPtr: f.viaPtr}
					field.Key = `"` + htmlEscape(name) + `":`
					fields = append(fields, field)
					if count[typeID(f.typ)] > 1 {
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}
				nextCount[typeID(ft)]++
				if nextCount[typeID(ft)] == 1 {
					next = append(next, candidate{typ: ft, index: index, viaPtr: f.viaPtr || ftPtr})
				}
			}
		}
	}
	sort.SliceStable(fields, func(i, j int) bool {
		a, b := fields[i], fields[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if len(a.Index) != len(b.Index) {
			return len(a.Index) < len(b.Index)
		}
		if a.tag != b.tag {
			return a.tag
		}
		return compareIndex(a.Index, b.Index) < 0
	})
	out := fields[:0:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		fi := fields[i]
		for advance = 1; i+advance < len(fields); advance++ {
			if fields[i+advance].Name != fi.Name {
				break
			}
		}
		if advance == 1 {
			out = append(out, fi)
			continue
		}
		dom := fields[i : i+advance]
		if len(dom) > 1 && len(dom[0].Index) == len(dom[1].Index) && dom[0].tag == dom[1].tag {
			continue
		}
		out = append(out, dom[0])
	}
	sort.SliceStable(out, func(i, j int) bool { return compareIndex(out[i].Index, out[j].Index) < 0 })
	return out
}

func compareIndex(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return len(a) - len(b)
}

func typeID(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() })
}

func parseTag(tag string) (string, string) {
	name, opts, _ := strings.Cut(tag, ",")
	return name, opts
}

func hasOpt(opts, name string) bool {
	for opts != "" {
		var o string
		o, opts, _ = strings.Cut(opts, ",")
		if o == name {
			return true
		}
	}
	return false
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}

func reflectTagGet(tag, key string) string {
	v, _ := reflectTagLookup(tag, key)
	return v
}

func reflectTagLookup(tag, key string) (string, bool) {
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := tag[:i]
		tag = tag[i+1:]
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := tag[:i+1]
		tag = tag[i+1:]
		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

const hexDigits = "0123456789abcdef"

func htmlEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' || c == '>' || c == '&' {
			b.WriteString(`\u00`)
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xF])
			continue
		}
		if c == 0xE2 && i+2 < len(s) && s[i+1] == 0x80 && s[i+2]&^1 == 0xA8 {
			b.WriteString(`\u202`)
			b.WriteByte(hexDigits[s[i+2]&0xF])
			i += 2
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// Problem describes why a type reachable from a json.Marshal operand cannot
// be encoded the way encoding/json encodes it.
type Problem struct {
	Type types.Type
	Path string
	Msg  string
}

// Check walks the types reachable when encoding t and reports the first one
// Goalchemy cannot encode exactly like encoding/json.
func Check(t types.Type) *Problem {
	return check(t, TypeString(t), map[string]bool{})
}

func check(t types.Type, path string, seen map[string]bool) *Problem {
	id := typeID(t)
	if seen[id] {
		return nil
	}
	seen[id] = true
	if Leaf(t) {
		return nil
	}
	switch u := t.Underlying().(type) {
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if p := checkTag(u, i, path); p != nil {
				return p
			}
		}
		for _, f := range Fields(t) {
			if p := check(f.Type, path+"."+f.Var.Name(), seen); p != nil {
				return p
			}
		}
		return nil
	case *types.Slice:
		if KindOf(t) == Bytes {
			return nil
		}
		return check(u.Elem(), path+"[]", seen)
	case *types.Array:
		return check(u.Elem(), path+"[]", seen)
	case *types.Map:
		if !ValidMapKey(u.Key()) {
			return &Problem{t, path, "map key type " + TypeString(u.Key()) + " is not supported"}
		}
		return check(u.Elem(), path+"[]", seen)
	case *types.Pointer:
		return check(u.Elem(), path, seen)
	case *types.Interface:
		return nil
	}
	if KindOf(t) == 0 {
		return &Problem{t, path, "type " + TypeString(t) + " is not supported"}
	}
	return nil
}

// checkTag rejects json tags whose meaning differs between encoding/json's
// releases or that std/encoding/json does not implement: names other than
// letters, digits and the punctuation encoding/json has always accepted,
// options other than omitempty and string, and tags on unexported fields.
func checkTag(st *types.Struct, i int, path string) *Problem {
	f := st.Field(i)
	tag, ok := reflectTagLookup(st.Tag(i), "json")
	if !ok || tag == "-" {
		return nil
	}
	at := path + "." + f.Name()
	if !f.Exported() && !f.Embedded() {
		return &Problem{f.Type(), at, "a json tag on an unexported field is not supported"}
	}
	name, opts := parseTag(tag)
	if name != "" && name != "-" && !isValidTag(name) {
		return &Problem{f.Type(), at, "json name " + strconv.Quote(name) + " is not supported; use letters, digits and punctuation other than quotes, backslash and comma"}
	}
	seen := map[string]bool{}
	for opts != "" {
		var o string
		o, opts, _ = strings.Cut(opts, ",")
		switch {
		case o == "":
		case o == "omitzero":
			return &Problem{f.Type(), at, "the omitzero option is not supported yet"}
		case o != "omitempty" && o != "string":
			return &Problem{f.Type(), at, "json option " + strconv.Quote(o) + " is not supported"}
		case seen[o]:
			return &Problem{f.Type(), at, "json option " + strconv.Quote(o) + " is repeated"}
		}
		seen[o] = true
	}
	ft := f.Type()
	if p, ok := types.Unalias(ft).(*types.Pointer); ok {
		ft = p.Elem()
	}
	if seen["string"] && IsNumber(ft) {
		return &Problem{f.Type(), at, "the string option on json.Number is not supported"}
	}
	return nil
}

// Dynamic reports whether a value of concrete type t stored in an interface
// is encoded by std/encoding/json the way encoding/json encodes it.
func Dynamic(t types.Type) bool {
	if types.IsInterface(t) {
		return true
	}
	if types.Implements(t, marshaler) || types.Implements(t, textMarshal) {
		return true
	}
	switch u := types.Unalias(t).(type) {
	case *types.Basic:
		return KindOf(u) != 0 || u.Kind() == types.UntypedNil
	case *types.Slice:
		if b, ok := types.Unalias(u.Elem()).(*types.Basic); ok && b.Kind() == types.Uint8 {
			return true
		}
		return isEmptyInterface(u.Elem())
	case *types.Map:
		k, ok := types.Unalias(u.Key()).(*types.Basic)
		return ok && k.Kind() == types.String && isEmptyInterface(u.Elem())
	}
	return false
}

func isEmptyInterface(t types.Type) bool {
	i, ok := types.Unalias(t).(*types.Interface)
	return ok && i.Empty()
}

// TypeString formats t as reflect.Type.String does.
func TypeString(t types.Type) string {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		o := t.Obj()
		if o.Pkg() == nil {
			return o.Name()
		}
		return o.Pkg().Name() + "." + o.Name()
	case *types.Basic:
		switch t.Kind() {
		case types.Byte:
			return "uint8"
		case types.Rune:
			return "int32"
		}
		return t.Name()
	case *types.Pointer:
		return "*" + TypeString(t.Elem())
	case *types.Slice:
		return "[]" + TypeString(t.Elem())
	case *types.Array:
		return "[" + strconv.FormatInt(t.Len(), 10) + "]" + TypeString(t.Elem())
	case *types.Map:
		return "map[" + TypeString(t.Key()) + "]" + TypeString(t.Elem())
	case *types.Chan:
		switch t.Dir() {
		case types.SendOnly:
			return "chan<- " + TypeString(t.Elem())
		case types.RecvOnly:
			return "<-chan " + TypeString(t.Elem())
		}
		return "chan " + TypeString(t.Elem())
	case *types.Signature:
		return "func" + signatureString(t)
	case *types.Interface:
		if t.NumMethods() == 0 {
			return "interface {}"
		}
		var ms []string
		for i := 0; i < t.NumMethods(); i++ {
			m := t.Method(i)
			ms = append(ms, m.Name()+signatureString(m.Signature()))
		}
		return "interface { " + strings.Join(ms, "; ") + " }"
	case *types.Struct:
		if t.NumFields() == 0 {
			return "struct {}"
		}
		var fs []string
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			s := TypeString(f.Type())
			if !f.Embedded() {
				s = f.Name() + " " + s
			}
			if tag := t.Tag(i); tag != "" {
				s += " " + strconv.Quote(tag)
			}
			fs = append(fs, s)
		}
		return "struct { " + strings.Join(fs, "; ") + " }"
	}
	return t.String()
}

func signatureString(s *types.Signature) string {
	var ps []string
	for i := 0; i < s.Params().Len(); i++ {
		p := TypeString(s.Params().At(i).Type())
		if s.Variadic() && i == s.Params().Len()-1 {
			p = "..." + strings.TrimPrefix(p, "[]")
		}
		ps = append(ps, p)
	}
	out := "(" + strings.Join(ps, ", ") + ")"
	switch s.Results().Len() {
	case 0:
	case 1:
		out += " " + TypeString(s.Results().At(0).Type())
	default:
		var rs []string
		for i := 0; i < s.Results().Len(); i++ {
			rs = append(rs, TypeString(s.Results().At(i).Type()))
		}
		out += " (" + strings.Join(rs, ", ") + ")"
	}
	return out
}

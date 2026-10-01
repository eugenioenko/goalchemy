package contracts

import (
	"fmt"
	"strconv"
	"strings"
)

// TypeExpr is a structured type reference used by contract signatures and
// test bindings. Type variables are specification metadata only.
type TypeExpr struct {
	Kind string // name, slice, array, map, pointer, chan
	Name string
	// Dir is "recv" or "send" for directional channels.
	Dir  string
	Len  int64
	Elem *TypeExpr
	Key  *TypeExpr
}

func (t *TypeExpr) String() string {
	switch t.Kind {
	case "slice":
		return "[]" + t.Elem.String()
	case "array":
		return fmt.Sprintf("[%d]%s", t.Len, t.Elem.String())
	case "map":
		return "map[" + t.Key.String() + "]" + t.Elem.String()
	case "pointer":
		return "*" + t.Elem.String()
	case "chan":
		switch t.Dir {
		case "recv":
			return "<-chan " + t.Elem.String()
		case "send":
			return "chan<- " + t.Elem.String()
		}
		return "chan " + t.Elem.String()
	}
	return t.Name
}

func ParseType(s string) (*TypeExpr, error) {
	t, rest, err := parseType(s)
	if err != nil {
		return nil, err
	}
	if rest != "" {
		return nil, fmt.Errorf("unexpected %q in type %q", rest, s)
	}
	return t, nil
}

func parseType(s string) (*TypeExpr, string, error) {
	switch {
	case strings.HasPrefix(s, "[]"):
		e, rest, err := parseType(s[2:])
		return &TypeExpr{Kind: "slice", Elem: e}, rest, err
	case strings.HasPrefix(s, "["):
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return nil, "", fmt.Errorf("unterminated array length in %q", s)
		}
		n, err := strconv.ParseInt(s[1:end], 10, 64)
		if err != nil || n < 0 {
			return nil, "", fmt.Errorf("invalid array length in %q", s)
		}
		e, rest, err := parseType(s[end+1:])
		return &TypeExpr{Kind: "array", Len: n, Elem: e}, rest, err
	case strings.HasPrefix(s, "map["):
		k, rest, err := parseType(s[4:])
		if err != nil {
			return nil, "", err
		}
		if !strings.HasPrefix(rest, "]") {
			return nil, "", fmt.Errorf("expected ] in map type %q", s)
		}
		v, rest, err := parseType(rest[1:])
		return &TypeExpr{Kind: "map", Key: k, Elem: v}, rest, err
	case strings.HasPrefix(s, "<-chan "):
		e, rest, err := parseType(s[7:])
		return &TypeExpr{Kind: "chan", Dir: "recv", Elem: e}, rest, err
	case strings.HasPrefix(s, "chan<- "):
		e, rest, err := parseType(s[7:])
		return &TypeExpr{Kind: "chan", Dir: "send", Elem: e}, rest, err
	case strings.HasPrefix(s, "chan "):
		e, rest, err := parseType(s[5:])
		return &TypeExpr{Kind: "chan", Elem: e}, rest, err
	case strings.HasPrefix(s, "func()"):
		return &TypeExpr{Kind: "name", Name: "func()"}, s[len("func()"):], nil
	case strings.HasPrefix(s, "struct{}"):
		return &TypeExpr{Kind: "name", Name: "struct{}"}, s[len("struct{}"):], nil
	case strings.HasPrefix(s, "*"):
		e, rest, err := parseType(s[1:])
		return &TypeExpr{Kind: "pointer", Elem: e}, rest, err
	}
	i := 0
	for i < len(s) && (s[i] == '_' || s[i] == '.' || s[i] >= 'a' && s[i] <= 'z' || s[i] >= 'A' && s[i] <= 'Z' || s[i] >= '0' && s[i] <= '9') {
		i++
	}
	if i == 0 {
		return nil, "", fmt.Errorf("expected type name in %q", s)
	}
	return &TypeExpr{Kind: "name", Name: s[:i]}, s[i:], nil
}

// Substitute replaces type variables using args.
func (t *TypeExpr) Substitute(args map[string]*TypeExpr) *TypeExpr {
	switch t.Kind {
	case "name":
		if a, ok := args[t.Name]; ok {
			return a
		}
		return t
	case "map":
		return &TypeExpr{Kind: "map", Key: t.Key.Substitute(args), Elem: t.Elem.Substitute(args)}
	}
	c := *t
	c.Elem = t.Elem.Substitute(args)
	return &c
}

var intRanges = map[string]struct {
	bits   uint
	signed bool
}{
	"int8": {8, true}, "int16": {16, true}, "int32": {32, true}, "int64": {64, true}, "int": {64, true},
	"uint8": {8, false}, "uint16": {16, false}, "uint32": {32, false}, "uint64": {64, false}, "uint": {64, false},
	"byte": {8, false}, "rune": {32, true},
}

// IntInfo reports the width and signedness of an integer type name,
// including external named integer types.
func IntInfo(name string) (bits uint, signed bool, ok bool) {
	if u, ok := externalInts[name]; ok {
		name = u
	}
	r, ok := intRanges[name]
	return r.bits, r.signed, ok
}

var concreteNames = map[string]bool{"bool": true, "string": true, "error": true, "any": true, "struct{}": true, "func()": true}

// Concrete reports whether a type expression contains only concrete names.
func (t *TypeExpr) Concrete() bool {
	switch t.Kind {
	case "name":
		_, isInt := intRanges[t.Name]
		// Package-qualified names denote external capability types.
		return isInt || concreteNames[t.Name] || strings.Contains(t.Name, ".")
	case "map":
		return t.Key.Concrete() && t.Elem.Concrete()
	}
	return t.Elem.Concrete()
}

// Comparable reports whether a concrete type is a valid Go map key.
func (t *TypeExpr) Comparable() bool {
	switch t.Kind {
	case "slice", "map":
		return false
	case "chan":
		return true
	case "array":
		return t.Elem.Comparable()
	}
	return true
}

// InFamily reports whether a concrete type satisfies a type-parameter family.
func InFamily(t *TypeExpr, family string) bool {
	switch family {
	case "any":
		return t.Concrete()
	case "type.integer":
		_, ok := intRanges[t.Name]
		return t.Kind == "name" && ok
	case "type.boolean":
		return t.Kind == "name" && t.Name == "bool"
	case "type.string":
		return t.Kind == "name" && t.Name == "string"
	case "type.slice":
		return t.Kind == "slice" && t.Concrete()
	case "type.map":
		return t.Kind == "map" && t.Concrete()
	case "type.array":
		return t.Kind == "array" && t.Concrete()
	case "type.pointer":
		return t.Kind == "pointer" && t.Concrete()
	case "type.channel":
		return t.Kind == "chan" && t.Concrete()
	case "type.error":
		return t.Kind == "name" && t.Name == "error"
	case "type.interface":
		return t.Kind == "name" && (t.Name == "any" || t.Name == "error")
	}
	return false
}

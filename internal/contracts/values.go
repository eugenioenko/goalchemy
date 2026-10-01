package contracts

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
)

// Value is a typed canonical test value. Integers are carried as decimal
// strings so that no host numeric precision is involved.
type Value struct {
	Type    *TypeExpr
	Kind    string // int, bool, string, slice, nil, array, map, view, ref, error
	Int     *big.Int
	Bool    bool
	Bytes   []byte
	Elems   []*Value
	Cap     *big.Int
	Entries [][2]*Value
	Name    string // view or ref target
	Lo, Hi  *big.Int
	Max     *big.Int
}

var canonicalDecimal = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

func ParseDecimal(s string) (*big.Int, error) {
	if !canonicalDecimal.MatchString(s) {
		return nil, fmt.Errorf("%q is not a canonical decimal integer", s)
	}
	v, _ := new(big.Int).SetString(s, 10)
	return v, nil
}

func checkIntRange(name string, v *big.Int) error {
	bits, signed, ok := IntInfo(name)
	if !ok {
		return fmt.Errorf("%s is not an integer type", name)
	}
	lo, hi := new(big.Int), new(big.Int).Lsh(big.NewInt(1), bits)
	if signed {
		hi.Rsh(hi, 1)
		lo.Neg(hi)
	}
	hi.Sub(hi, big.NewInt(1))
	if v.Cmp(lo) < 0 || v.Cmp(hi) > 0 {
		return fmt.Errorf("%s is outside the range of %s", v, name)
	}
	return nil
}

// Scope maps let-binding names to their types for view and ref checks.
type Scope map[string]*TypeExpr

func DecodeValue(raw json.RawMessage, t *TypeExpr, scope Scope) (*Value, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return decodeValue(v, t, scope)
}

func decodeValue(v any, t *TypeExpr, scope Scope) (*Value, error) {
	out := &Value{Type: t}
	if s, ok := v.(string); ok {
		if t.Kind != "name" {
			return nil, fmt.Errorf("scalar %q given for %s", s, t)
		}
		switch t.Name {
		case "bool":
			if s != "true" && s != "false" {
				return nil, fmt.Errorf("Boolean must be \"true\" or \"false\", got %q", s)
			}
			out.Kind, out.Bool = "bool", s == "true"
			return out, nil
		case "string", "error", "any":
			return nil, fmt.Errorf("%s values need an object form", t.Name)
		}
		n, err := ParseDecimal(s)
		if err != nil {
			return nil, err
		}
		if err := checkIntRange(t.Name, n); err != nil {
			return nil, err
		}
		out.Kind, out.Int = "int", n
		return out, nil
	}
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return nil, fmt.Errorf("value must be a string or a tagged object")
	}
	num := func(key string) (*big.Int, error) {
		s, present := m[key]
		if !present {
			return nil, nil
		}
		str, ok := s.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be a decimal string", key)
		}
		n, err := ParseDecimal(str)
		if err == nil && n.Sign() < 0 {
			err = fmt.Errorf("%s must be non-negative", key)
		}
		return n, err
	}
	switch {
	case m["str"] != nil:
		if t.Kind != "name" || t.Name != "string" {
			return nil, fmt.Errorf("string value given for %s", t)
		}
		out.Kind, out.Bytes = "string", []byte(m["str"].(string))
	case m["hex"] != nil:
		if t.Kind != "name" || t.Name != "string" {
			return nil, fmt.Errorf("string value given for %s", t)
		}
		b, err := hex.DecodeString(m["hex"].(string))
		if err != nil {
			return nil, err
		}
		out.Kind, out.Bytes = "string", b
	case m["nil"] != nil:
		switch {
		case t.Kind == "slice", t.Kind == "map", t.Kind == "pointer",
			t.Kind == "name" && (t.Name == "error" || t.Name == "any"):
		default:
			return nil, fmt.Errorf("nil is not a value of %s", t)
		}
		out.Kind = "nil"
	case m["slice"] != nil:
		if t.Kind != "slice" {
			return nil, fmt.Errorf("slice value given for %s", t)
		}
		items, _ := m["slice"].([]any)
		out.Kind = "slice"
		for i, it := range items {
			e, err := decodeValue(it, t.Elem, scope)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			out.Elems = append(out.Elems, e)
		}
		c, err := num("cap")
		if err != nil {
			return nil, err
		}
		if c != nil && c.Cmp(big.NewInt(int64(len(items)))) < 0 {
			return nil, fmt.Errorf("cap %s is less than length %d", c, len(items))
		}
		out.Cap = c
	case m["array"] != nil:
		if t.Kind != "array" {
			return nil, fmt.Errorf("array value given for %s", t)
		}
		items, _ := m["array"].([]any)
		if int64(len(items)) != t.Len {
			return nil, fmt.Errorf("array has %d elements, want %d", len(items), t.Len)
		}
		out.Kind = "array"
		for i, it := range items {
			e, err := decodeValue(it, t.Elem, scope)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			out.Elems = append(out.Elems, e)
		}
	case m["map"] != nil:
		if t.Kind != "map" {
			return nil, fmt.Errorf("map value given for %s", t)
		}
		out.Kind = "map"
		items, _ := m["map"].([]any)
		for i, it := range items {
			ent, _ := it.(map[string]any)
			k, err := decodeValue(ent["key"], t.Key, scope)
			if err != nil {
				return nil, fmt.Errorf("entry %d key: %w", i, err)
			}
			val, err := decodeValue(ent["value"], t.Elem, scope)
			if err != nil {
				return nil, fmt.Errorf("entry %d value: %w", i, err)
			}
			out.Entries = append(out.Entries, [2]*Value{k, val})
		}
	case m["view"] != nil:
		if t.Kind != "slice" {
			return nil, fmt.Errorf("view given for %s", t)
		}
		name := m["view"].(string)
		bt, ok := scope[name]
		if !ok {
			return nil, fmt.Errorf("view of unknown binding %q", name)
		}
		if bt.String() != t.String() {
			return nil, fmt.Errorf("view of %s binding %q as %s", bt, name, t)
		}
		out.Kind, out.Name = "view", name
		var err error
		if out.Lo, err = num("lo"); err != nil {
			return nil, err
		}
		if out.Hi, err = num("hi"); err != nil {
			return nil, err
		}
		if out.Max, err = num("max"); err != nil {
			return nil, err
		}
	case m["error"] != nil:
		if t.Kind != "name" || t.Name != "error" {
			return nil, fmt.Errorf("error value given for %s", t)
		}
		msg, ok := m["error"].(string)
		if !ok {
			return nil, fmt.Errorf("error message must be a string")
		}
		out.Kind, out.Bytes = "error", []byte(msg)
	case m["ref"] != nil:
		name := m["ref"].(string)
		bt, ok := scope[name]
		if !ok {
			return nil, fmt.Errorf("reference to unknown binding %q", name)
		}
		if bt.String() != t.String() {
			return nil, fmt.Errorf("reference to %s binding %q as %s", bt, name, t)
		}
		out.Kind, out.Name = "ref", name
	default:
		return nil, fmt.Errorf("unknown value form")
	}
	return out, nil
}

// Encode renders a value in the canonical JSON wire form.
func (v *Value) Encode() any {
	switch v.Kind {
	case "int":
		return v.Int.String()
	case "bool":
		if v.Bool {
			return "true"
		}
		return "false"
	case "string":
		return map[string]any{"hex": hex.EncodeToString(v.Bytes)}
	case "nil":
		return map[string]any{"nil": true}
	case "slice", "array":
		items := make([]any, len(v.Elems))
		for i, e := range v.Elems {
			items[i] = e.Encode()
		}
		if v.Kind == "array" {
			return map[string]any{"array": items}
		}
		m := map[string]any{"slice": items}
		if v.Cap != nil {
			m["cap"] = v.Cap.String()
		}
		return m
	case "map":
		items := make([]any, len(v.Entries))
		for i, e := range v.Entries {
			items[i] = map[string]any{"key": e[0].Encode(), "value": e[1].Encode()}
		}
		return map[string]any{"map": items}
	case "view":
		m := map[string]any{"view": v.Name}
		for k, n := range map[string]*big.Int{"lo": v.Lo, "hi": v.Hi, "max": v.Max} {
			if n != nil {
				m[k] = n.String()
			}
		}
		return m
	case "ref":
		return map[string]any{"ref": v.Name}
	case "error":
		return map[string]any{"error": string(v.Bytes)}
	}
	return nil
}

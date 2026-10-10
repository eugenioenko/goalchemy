// Package jsonvalue parses, inspects, builds and encodes JSON documents of
// any shape as a tree of Values, without declaring Go types.
//
// Numbers keep their exact source text, so integers beyond 2^53 and long
// decimals round-trip unchanged. Object members keep their order, and
// duplicate member names are rejected. Parsing and encoding are bounded by
// Limits. Accessors never panic: a missing member, an out-of-range index or a
// value of another kind yields the zero result.
package jsonvalue

import (
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

// Kind identifies the JSON type of a Value.
type Kind int

const (
	KindInvalid Kind = iota
	KindNull
	KindBool
	KindNumber
	KindString
	KindArray
	KindObject
)

var kindNames = []string{"invalid", "null", "bool", "number", "string", "array", "object"}

func (k Kind) String() string {
	if k < 0 || int(k) >= len(kindNames) {
		return "Kind(" + strconv.Itoa(int(k)) + ")"
	}
	return kindNames[k]
}

// Value is one JSON value. The zero Value is invalid: it is what lookups of
// missing members and out-of-range indexes return.
type Value struct {
	kind  Kind
	text  string
	b     bool
	keys  []string
	elems []Value
}

// Limits bounds the work done by parsing and encoding. Depth counts nested
// arrays and objects, Nodes counts values, and StringBytes bounds each string
// after unescaping. Depth must not exceed MaxDepth.
type Limits struct {
	Bytes       int
	Depth       int
	Nodes       int
	StringBytes int
}

// MaxDepth is the largest nesting depth Limits may allow.
const MaxDepth = 128

// DefaultLimits returns the limits Parse and Encode use: 16 MiB of input or
// output, nesting depth 128, 1Mi values and 16 MiB per string.
func DefaultLimits() Limits { return Limits{16 << 20, MaxDepth, 1 << 20, 16 << 20} }

func (l Limits) valid() bool {
	return l.Bytes > 0 && l.Depth > 0 && l.Depth <= MaxDepth && l.Nodes > 0 && l.StringBytes > 0
}

// SyntaxError describes invalid JSON input. Offset is the byte offset at
// which the error was detected.
type SyntaxError struct {
	Msg    string
	Offset int
}

func (e *SyntaxError) Error() string {
	return "jsonvalue: " + e.Msg + " at offset " + strconv.Itoa(e.Offset)
}

// LimitError reports that parsing or encoding exceeded one of the Limits.
// Limit names it: "bytes", "depth", "nodes", "string bytes" or "limits" when
// the Limits themselves are invalid.
type LimitError struct {
	Limit string
}

func (e *LimitError) Error() string {
	if e.Limit == "limits" {
		return "jsonvalue: invalid limits"
	}
	return "jsonvalue: " + e.Limit + " limit exceeded"
}

// InvalidValueError reports a Value that cannot be encoded: a number that is
// not valid JSON (including NaN and infinities), an object with duplicate
// member names, or an invalid Value.
type InvalidValueError struct {
	Msg string
}

func (e *InvalidValueError) Error() string { return "jsonvalue: " + e.Msg }

// Null returns the JSON null value.
func Null() Value { return Value{kind: KindNull} }

// Bool returns a JSON boolean.
func Bool(b bool) Value { return Value{kind: KindBool, b: b} }

// String returns a JSON string. Invalid UTF-8 is encoded as U+FFFD.
func String(s string) Value { return Value{kind: KindString, text: s} }

// Int returns a JSON number holding n exactly.
func Int(n int64) Value { return Value{kind: KindNumber, text: strconv.FormatInt(n, 10)} }

// Uint returns a JSON number holding n exactly.
func Uint(n uint64) Value { return Value{kind: KindNumber, text: strconv.FormatUint(n, 10)} }

// Float returns a JSON number formatted as encoding/json formats a float64:
// the shortest representation that round-trips, in exponent form below 1e-6
// and from 1e21. NaN and infinities produce a Value that Encode rejects.
func Float(f float64) Value {
	if f != f {
		return Value{kind: KindNumber, text: "NaN"}
	}
	if f > 1.7976931348623157e308 {
		return Value{kind: KindNumber, text: "+Inf"}
	}
	if f < -1.7976931348623157e308 {
		return Value{kind: KindNumber, text: "-Inf"}
	}
	abs := f
	if abs < 0 {
		abs = -abs
	}
	format := byte('f')
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	b := strconv.AppendFloat(nil, f, format, -1, 64)
	if format == 'e' {
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}
	return Value{kind: KindNumber, text: string(b)}
}

// Number returns a JSON number with the given source text, such as
// "12345678901234567890" or "1.50e+3". Encode rejects text that is not a
// valid JSON number.
func Number(text string) Value { return Value{kind: KindNumber, text: text} }

// Array returns a JSON array of elems.
func Array(elems ...Value) Value {
	return Value{kind: KindArray, elems: append([]Value{}, elems...)}
}

// Object returns a JSON object from alternating string keys and Value
// values, in order. It panics if the arguments do not alternate that way.
func Object(keyvals ...any) Value {
	if len(keyvals)%2 != 0 {
		panic("jsonvalue: Object needs alternating keys and values")
	}
	v := Value{kind: KindObject, keys: []string{}, elems: []Value{}}
	for i := 0; i < len(keyvals); i += 2 {
		k, ok := keyvals[i].(string)
		if !ok {
			panic("jsonvalue: Object key " + strconv.Itoa(i/2) + " is not a string")
		}
		x, ok := keyvals[i+1].(Value)
		if !ok {
			panic("jsonvalue: Object value for " + strconv.Quote(k) + " is not a Value")
		}
		v.keys = append(v.keys, k)
		v.elems = append(v.elems, x)
	}
	return v
}

// Set returns a copy of the object v with member key set to x, replacing an
// existing member in place or appending a new one. If v is not an object,
// Set returns a new object holding only that member.
func (v Value) Set(key string, x Value) Value {
	if v.kind != KindObject {
		return Value{kind: KindObject, keys: []string{key}, elems: []Value{x}}
	}
	out := Value{kind: KindObject, keys: append([]string{}, v.keys...), elems: append([]Value{}, v.elems...)}
	for i, k := range out.keys {
		if k == key {
			out.elems[i] = x
			return out
		}
	}
	out.keys = append(out.keys, key)
	out.elems = append(out.elems, x)
	return out
}

// Append returns a copy of the array v with elems added at the end. If v is
// not an array, Append returns a new array holding only elems.
func (v Value) Append(elems ...Value) Value {
	if v.kind != KindArray {
		return Array(elems...)
	}
	out := append([]Value{}, v.elems...)
	return Value{kind: KindArray, elems: append(out, elems...)}
}

// Kind returns the JSON type of v.
func (v Value) Kind() Kind { return v.kind }

// Exists reports whether v is a value rather than the result of a failed
// lookup.
func (v Value) Exists() bool { return v.kind != KindInvalid }

// IsNull reports whether v is JSON null.
func (v Value) IsNull() bool { return v.kind == KindNull }

// Lookup returns the member key of the object v.
func (v Value) Lookup(key string) (Value, bool) {
	if v.kind != KindObject {
		return Value{}, false
	}
	for i, k := range v.keys {
		if k == key {
			return v.elems[i], true
		}
	}
	return Value{}, false
}

// Get returns the member key of the object v, or the zero Value.
func (v Value) Get(key string) Value {
	x, _ := v.Lookup(key)
	return x
}

// Has reports whether the object v has a member key.
func (v Value) Has(key string) bool {
	_, ok := v.Lookup(key)
	return ok
}

// Index returns element i of the array v, or the zero Value.
func (v Value) Index(i int) Value {
	if v.kind != KindArray || i < 0 || i >= len(v.elems) {
		return Value{}
	}
	return v.elems[i]
}

// Len returns the number of elements of an array or members of an object,
// and 0 for other kinds.
func (v Value) Len() int {
	if v.kind != KindArray && v.kind != KindObject {
		return 0
	}
	return len(v.elems)
}

// Keys returns the member names of the object v in order.
func (v Value) Keys() []string {
	if v.kind != KindObject {
		return nil
	}
	return append([]string{}, v.keys...)
}

// AsString returns the content of a JSON string.
func (v Value) AsString() (string, bool) {
	if v.kind != KindString {
		return "", false
	}
	return v.text, true
}

// String returns the content of a JSON string, the compact encoding of any
// other value, and "" for the zero Value or a Value that cannot be encoded.
// Use AsString to require a string.
func (v Value) String() string {
	switch v.kind {
	case KindInvalid:
		return ""
	case KindString:
		return v.text
	}
	b, err := Encode(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// AsBool returns the value of a JSON boolean.
func (v Value) AsBool() (bool, bool) {
	if v.kind != KindBool {
		return false, false
	}
	return v.b, true
}

// Bool returns the value of a JSON boolean, or false.
func (v Value) Bool() bool {
	b, _ := v.AsBool()
	return b
}

// Number returns the source text of a JSON number, or "".
func (v Value) Number() string {
	if v.kind != KindNumber {
		return ""
	}
	return v.text
}

func (v Value) integer() (string, bool) {
	if v.kind != KindNumber || !validNumber(v.text) {
		return "", false
	}
	for i := 0; i < len(v.text); i++ {
		if c := v.text[i]; c == '.' || c == 'e' || c == 'E' {
			return "", false
		}
	}
	return v.text, true
}

// AsInt64 returns a JSON number written as an integer that fits in int64.
// Numbers with a fraction or exponent, such as 1.0 or 1e3, are rejected.
func (v Value) AsInt64() (int64, bool) {
	s, ok := v.integer()
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

// Int64 returns the result of AsInt64, or 0.
func (v Value) Int64() int64 {
	n, _ := v.AsInt64()
	return n
}

// AsUint64 returns a JSON number written as a nonnegative integer that fits
// in uint64.
func (v Value) AsUint64() (uint64, bool) {
	s, ok := v.integer()
	if !ok || s[0] == '-' {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil
}

// Uint64 returns the result of AsUint64, or 0.
func (v Value) Uint64() uint64 {
	n, _ := v.AsUint64()
	return n
}

// AsFloat64 returns the float64 nearest to a JSON number. Numbers beyond the
// float64 range are rejected.
func (v Value) AsFloat64() (float64, bool) {
	if v.kind != KindNumber || !validNumber(v.text) {
		return 0, false
	}
	f, err := strconv.ParseFloat(v.text, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// Float64 returns the result of AsFloat64, or 0.
func (v Value) Float64() float64 {
	f, _ := v.AsFloat64()
	return f
}

// Parse parses one JSON document with DefaultLimits.
func Parse(data []byte) (Value, error) { return ParseWithLimits(data, DefaultLimits()) }

// ParseWithLimits parses one JSON document. Surrounding whitespace is
// allowed; any other trailing input is an error. Invalid UTF-8 and unpaired
// surrogate escapes in strings become U+FFFD, as in encoding/json.
func ParseWithLimits(data []byte, l Limits) (Value, error) {
	if !l.valid() {
		return Value{}, &LimitError{"limits"}
	}
	if len(data) > l.Bytes {
		return Value{}, &LimitError{"bytes"}
	}
	p := parser{data: data, limits: l}
	v, err := p.value(0)
	if err != nil {
		return Value{}, err
	}
	p.space()
	if p.pos != len(data) {
		return Value{}, p.fail("unexpected data after top-level value")
	}
	return v, nil
}

type parser struct {
	data   []byte
	pos    int
	nodes  int
	limits Limits
}

func (p *parser) fail(msg string) error { return &SyntaxError{msg, p.pos} }

func (p *parser) space() {
	for p.pos < len(p.data) {
		b := p.data[p.pos]
		if b != ' ' && b != '\n' && b != '\r' && b != '\t' {
			return
		}
		p.pos++
	}
}

func (p *parser) value(depth int) (Value, error) {
	p.space()
	p.nodes++
	if p.nodes > p.limits.Nodes {
		return Value{}, &LimitError{"nodes"}
	}
	if p.pos >= len(p.data) {
		return Value{}, p.fail("unexpected end of input")
	}
	switch c := p.data[p.pos]; c {
	case '"':
		s, err := p.str()
		if err != nil {
			return Value{}, err
		}
		return String(s), nil
	case '{', '[':
		if depth >= p.limits.Depth {
			return Value{}, &LimitError{"depth"}
		}
		return p.container(c, depth)
	case 'n':
		return p.literal("null", Null())
	case 't':
		return p.literal("true", Bool(true))
	case 'f':
		return p.literal("false", Bool(false))
	}
	end := numberEnd(p.data, p.pos)
	if end == p.pos {
		return Value{}, p.fail("invalid character " + strconv.QuoteRune(rune(p.data[p.pos])) + " looking for beginning of value")
	}
	v := Number(string(p.data[p.pos:end]))
	p.pos = end
	return v, nil
}

func (p *parser) literal(word string, v Value) (Value, error) {
	for i := 0; i < len(word); i++ {
		if p.pos >= len(p.data) {
			return Value{}, p.fail("unexpected end of input")
		}
		if p.data[p.pos] != word[i] {
			return Value{}, p.fail("invalid character " + strconv.QuoteRune(rune(p.data[p.pos])) + " in literal " + word)
		}
		p.pos++
	}
	return v, nil
}

func (p *parser) container(open byte, depth int) (Value, error) {
	p.pos++
	v := Value{kind: KindArray, elems: []Value{}}
	close := byte(']')
	var seen map[string]bool
	if open == '{' {
		v.kind = KindObject
		v.keys = []string{}
		close = '}'
		seen = map[string]bool{}
	}
	p.space()
	if p.pos < len(p.data) && p.data[p.pos] == close {
		p.pos++
		return v, nil
	}
	for {
		if v.kind == KindObject {
			p.space()
			if p.pos >= len(p.data) {
				return Value{}, p.fail("unexpected end of input")
			}
			if p.data[p.pos] != '"' {
				return Value{}, p.fail("expected string for object member name")
			}
			start := p.pos
			k, err := p.str()
			if err != nil {
				return Value{}, err
			}
			if seen[k] {
				return Value{}, &SyntaxError{"duplicate object member " + strconv.Quote(k), start}
			}
			seen[k] = true
			v.keys = append(v.keys, k)
			p.space()
			if p.pos >= len(p.data) {
				return Value{}, p.fail("unexpected end of input")
			}
			if p.data[p.pos] != ':' {
				return Value{}, p.fail("expected ':' after object member name")
			}
			p.pos++
		}
		x, err := p.value(depth + 1)
		if err != nil {
			return Value{}, err
		}
		v.elems = append(v.elems, x)
		p.space()
		if p.pos >= len(p.data) {
			return Value{}, p.fail("unexpected end of input")
		}
		c := p.data[p.pos]
		if c == close {
			p.pos++
			return v, nil
		}
		if c != ',' {
			if close == '}' {
				return Value{}, p.fail("expected ',' or '}' after object member")
			}
			return Value{}, p.fail("expected ',' or ']' after array element")
		}
		p.pos++
	}
}

func numberEnd(b []byte, i int) int {
	start := i
	if i < len(b) && b[i] == '-' {
		i++
	}
	if i >= len(b) {
		return start
	}
	if b[i] == '0' {
		i++
	} else {
		if b[i] < '1' || b[i] > '9' {
			return start
		}
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
		}
	}
	if i < len(b) && b[i] == '.' {
		i++
		s := i
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
		}
		if i == s {
			return start
		}
	}
	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		i++
		if i < len(b) && (b[i] == '+' || b[i] == '-') {
			i++
		}
		s := i
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
		}
		if s == i {
			return start
		}
	}
	return i
}

func validNumber(s string) bool {
	return len(s) > 0 && numberEnd([]byte(s), 0) == len(s)
}

func unhex(b byte) rune {
	switch {
	case '0' <= b && b <= '9':
		return rune(b - '0')
	case 'a' <= b && b <= 'f':
		return rune(b-'a') + 10
	case 'A' <= b && b <= 'F':
		return rune(b-'A') + 10
	}
	return -1
}

func (p *parser) hex4(at int) rune {
	if len(p.data)-at < 6 || p.data[at] != '\\' || p.data[at+1] != 'u' {
		return -1
	}
	var r rune
	for _, b := range p.data[at+2 : at+6] {
		h := unhex(b)
		if h < 0 {
			return -1
		}
		r = r*16 + h
	}
	return r
}

func (p *parser) str() (string, error) {
	p.pos++
	out := []byte{}
	for {
		start := p.pos
		for p.pos < len(p.data) {
			c := p.data[p.pos]
			if c == '"' || c == '\\' || c < 0x20 || c >= utf8.RuneSelf {
				break
			}
			p.pos++
		}
		out = append(out, p.data[start:p.pos]...)
		if len(out) > p.limits.StringBytes {
			return "", &LimitError{"string bytes"}
		}
		if p.pos >= len(p.data) {
			return "", p.fail("unexpected end of input in string")
		}
		c := p.data[p.pos]
		switch {
		case c == '"':
			p.pos++
			return string(out), nil
		case c < 0x20:
			return "", p.fail("invalid character " + strconv.QuoteRune(rune(c)) + " in string literal")
		case c >= utf8.RuneSelf:
			r, size := utf8.DecodeRune(p.data[p.pos:])
			if r == utf8.RuneError && size == 1 {
				out = utf8.AppendRune(out, utf8.RuneError)
			} else {
				out = append(out, p.data[p.pos:p.pos+size]...)
			}
			p.pos += size
		default:
			if p.pos+1 >= len(p.data) {
				return "", p.fail("unexpected end of input in string escape")
			}
			e := p.data[p.pos+1]
			switch e {
			case '"', '\\', '/':
				out = append(out, e)
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'u':
				r := p.hex4(p.pos)
				if r < 0 {
					return "", p.fail("invalid \\u escape in string literal")
				}
				p.pos += 6
				if 0xD800 <= r && r < 0xE000 {
					if r < 0xDC00 {
						if r2 := p.hex4(p.pos); 0xDC00 <= r2 && r2 < 0xE000 {
							p.pos += 6
							out = utf8.AppendRune(out, (r-0xD800)<<10|(r2-0xDC00)+0x10000)
							continue
						}
					}
					r = utf8.RuneError
				}
				out = utf8.AppendRune(out, r)
				continue
			default:
				return "", p.fail("invalid character " + strconv.QuoteRune(rune(e)) + " in string escape code")
			}
			p.pos += 2
		}
		if len(out) > p.limits.StringBytes {
			return "", &LimitError{"string bytes"}
		}
	}
}

// Encode returns the compact JSON encoding of v with DefaultLimits.
func Encode(v Value) ([]byte, error) { return EncodeWithLimits(v, DefaultLimits()) }

// MarshalJSON returns Encode(v), so a Value encodes as itself when it is
// passed to, or is a field of a value passed to, json.Marshal.
func (v Value) MarshalJSON() ([]byte, error) { return Encode(v) }

// UnmarshalJSON sets *v to Parse(data), so a Value can be the target of, or
// a field of the target of, json.Unmarshal.
func (v *Value) UnmarshalJSON(data []byte) error {
	x, err := Parse(data)
	if err != nil {
		return err
	}
	*v = x
	return nil
}

// EncodeWithLimits returns the compact JSON encoding of v. Strings are
// escaped exactly as encoding/json.Marshal escapes them, including <, > and
// & as <, > and &, U+2028 and U+2029, and invalid UTF-8 as
// �. Numbers are written from their source text.
func EncodeWithLimits(v Value, l Limits) ([]byte, error) {
	if !l.valid() {
		return nil, &LimitError{"limits"}
	}
	e := encoder{limits: l}
	if err := e.value(v, 0); err != nil {
		return nil, err
	}
	if len(e.out) > l.Bytes {
		return nil, &LimitError{"bytes"}
	}
	return e.out, nil
}

type encoder struct {
	out    []byte
	nodes  int
	limits Limits
}

func (e *encoder) value(v Value, depth int) error {
	e.nodes++
	if e.nodes > e.limits.Nodes {
		return &LimitError{"nodes"}
	}
	if len(e.out) > e.limits.Bytes {
		return &LimitError{"bytes"}
	}
	switch v.kind {
	case KindNull:
		e.out = append(e.out, "null"...)
	case KindBool:
		if v.b {
			e.out = append(e.out, "true"...)
		} else {
			e.out = append(e.out, "false"...)
		}
	case KindNumber:
		if !validNumber(v.text) {
			if v.text == "NaN" || v.text == "+Inf" || v.text == "-Inf" {
				return &InvalidValueError{"unsupported value: " + v.text}
			}
			return &InvalidValueError{"invalid number literal " + strconv.Quote(v.text)}
		}
		e.out = append(e.out, v.text...)
	case KindString:
		return e.str(v.text)
	case KindArray, KindObject:
		if depth >= e.limits.Depth {
			return &LimitError{"depth"}
		}
		return e.container(v, depth)
	default:
		return &InvalidValueError{"cannot encode an invalid Value"}
	}
	return nil
}

func (e *encoder) container(v Value, depth int) error {
	if v.kind == KindArray {
		e.out = append(e.out, '[')
		for i, x := range v.elems {
			if i > 0 {
				e.out = append(e.out, ',')
			}
			if err := e.value(x, depth+1); err != nil {
				return err
			}
		}
		e.out = append(e.out, ']')
		return nil
	}
	e.out = append(e.out, '{')
	seen := map[string]bool{}
	for i, k := range v.keys {
		if seen[k] {
			return &InvalidValueError{"duplicate object member " + strconv.Quote(k)}
		}
		seen[k] = true
		if i > 0 {
			e.out = append(e.out, ',')
		}
		if err := e.str(k); err != nil {
			return err
		}
		e.out = append(e.out, ':')
		if err := e.value(v.elems[i], depth+1); err != nil {
			return err
		}
	}
	e.out = append(e.out, '}')
	return nil
}

const hexDigits = "0123456789abcdef"

func (e *encoder) str(s string) error {
	if len(s) > e.limits.StringBytes {
		return &LimitError{"string bytes"}
	}
	b := []byte(s)
	e.out = append(e.out, '"')
	start := 0
	for i := 0; i < len(b); {
		c := b[i]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' && c != '<' && c != '>' && c != '&' {
				i++
				continue
			}
			e.out = append(e.out, b[start:i]...)
			switch c {
			case '"', '\\':
				e.out = append(e.out, '\\', c)
			case '\b':
				e.out = append(e.out, '\\', 'b')
			case '\f':
				e.out = append(e.out, '\\', 'f')
			case '\n':
				e.out = append(e.out, '\\', 'n')
			case '\r':
				e.out = append(e.out, '\\', 'r')
			case '\t':
				e.out = append(e.out, '\\', 't')
			default:
				e.out = append(e.out, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xF])
			}
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			e.out = append(e.out, b[start:i]...)
			e.out = append(e.out, `�`...)
			i++
			start = i
			continue
		}
		if r == ' ' || r == ' ' {
			e.out = append(e.out, b[start:i]...)
			e.out = append(e.out, '\\', 'u', '2', '0', '2', hexDigits[r&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	e.out = append(e.out, b[start:]...)
	e.out = append(e.out, '"')
	return nil
}

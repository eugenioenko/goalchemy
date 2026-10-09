package slog

import (
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/time"
)

// A Value can represent any Go value without allocating for the common
// kinds.
type Value struct {
	kind  Kind
	num   uint64
	float float64
	str   string
	t     time.Time
	group []Attr
	any   any
}

// Kind is the kind of a Value.
type Kind int

// The kinds of Values.
const (
	KindAny Kind = iota
	KindBool
	KindDuration
	KindFloat64
	KindInt64
	KindString
	KindTime
	KindUint64
	KindGroup
	KindLogValuer
)

var kindStrings = []string{"Any", "Bool", "Duration", "Float64", "Int64", "String", "Time", "Uint64", "Group", "LogValuer"}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindStrings) {
		return kindStrings[k]
	}
	return "<unknown slog.Kind>"
}

// Kind returns v's Kind.
func (v Value) Kind() Kind { return v.kind }

// StringValue returns a Value for a string.
func StringValue(value string) Value { return Value{kind: KindString, str: value} }

// IntValue returns a Value for an int.
func IntValue(v int) Value { return Int64Value(int64(v)) }

// Int64Value returns a Value for an int64.
func Int64Value(v int64) Value { return Value{kind: KindInt64, num: uint64(v)} }

// Uint64Value returns a Value for a uint64.
func Uint64Value(v uint64) Value { return Value{kind: KindUint64, num: v} }

// Float64Value returns a Value for a floating-point number.
func Float64Value(v float64) Value { return Value{kind: KindFloat64, float: v} }

// BoolValue returns a Value for a bool.
func BoolValue(v bool) Value {
	u := uint64(0)
	if v {
		u = 1
	}
	return Value{kind: KindBool, num: u}
}

// TimeValue returns a Value for a time.Time.
func TimeValue(v time.Time) Value { return Value{kind: KindTime, t: v} }

// DurationValue returns a Value for a time.Duration.
func DurationValue(v time.Duration) Value {
	return Value{kind: KindDuration, num: uint64(v.Nanoseconds())}
}

// GroupValue returns a new Value for a list of Attrs, omitting empty groups.
func GroupValue(as ...Attr) Value {
	if n := countEmptyGroups(as); n > 0 {
		as2 := make([]Attr, 0, len(as)-n)
		for _, a := range as {
			if !a.Value.isEmptyGroup() {
				as2 = append(as2, a)
			}
		}
		as = as2
	}
	return Value{kind: KindGroup, group: as}
}

func countEmptyGroups(as []Attr) int {
	n := 0
	for _, a := range as {
		if a.Value.isEmptyGroup() {
			n++
		}
	}
	return n
}

// AnyValue returns a Value for v, using the kind that matches v's type.
func AnyValue(v any) Value {
	switch v := v.(type) {
	case string:
		return StringValue(v)
	case int:
		return Int64Value(int64(v))
	case uint:
		return Uint64Value(uint64(v))
	case int64:
		return Int64Value(v)
	case uint64:
		return Uint64Value(v)
	case bool:
		return BoolValue(v)
	case time.Duration:
		return DurationValue(v)
	case time.Time:
		return TimeValue(v)
	case uint8:
		return Uint64Value(uint64(v))
	case uint16:
		return Uint64Value(uint64(v))
	case uint32:
		return Uint64Value(uint64(v))
	case int8:
		return Int64Value(int64(v))
	case int16:
		return Int64Value(int64(v))
	case int32:
		return Int64Value(int64(v))
	case float64:
		return Float64Value(v)
	case float32:
		return Float64Value(float64(v))
	case []Attr:
		return GroupValue(v...)
	case Value:
		return v
	case LogValuer:
		return Value{kind: KindLogValuer, any: v}
	default:
		return Value{kind: KindAny, any: v}
	}
}

// Any returns v's value as an any.
func (v Value) Any() any {
	switch v.kind {
	case KindAny, KindLogValuer:
		return v.any
	case KindGroup:
		return v.group
	case KindInt64:
		return int64(v.num)
	case KindUint64:
		return v.num
	case KindFloat64:
		return v.float
	case KindString:
		return v.str
	case KindBool:
		return v.num == 1
	case KindDuration:
		return time.Duration(int64(v.num))
	case KindTime:
		return v.t
	default:
		panic("bad kind: " + v.kind.String())
	}
}

// String returns v's value as a string, formatted like fmt.Sprint.
func (v Value) String() string {
	if v.kind == KindString {
		return v.str
	}
	return string(v.append(nil))
}

func (v Value) mustBe(k Kind) {
	if v.kind != k {
		panic("Value kind is " + v.kind.String() + ", not " + k.String())
	}
}

// Int64 returns v's value as an int64. It panics if v is not a signed integer.
func (v Value) Int64() int64 {
	v.mustBe(KindInt64)
	return int64(v.num)
}

// Uint64 returns v's value as a uint64. It panics if v is not an unsigned integer.
func (v Value) Uint64() uint64 {
	v.mustBe(KindUint64)
	return v.num
}

// Bool returns v's value as a bool. It panics if v is not a bool.
func (v Value) Bool() bool {
	v.mustBe(KindBool)
	return v.num == 1
}

// Duration returns v's value as a time.Duration. It panics if v is not a time.Duration.
func (v Value) Duration() time.Duration {
	v.mustBe(KindDuration)
	return time.Duration(int64(v.num))
}

// Float64 returns v's value as a float64. It panics if v is not a float64.
func (v Value) Float64() float64 {
	v.mustBe(KindFloat64)
	return v.float
}

// Time returns v's value as a time.Time. It panics if v is not a time.Time.
func (v Value) Time() time.Time {
	v.mustBe(KindTime)
	return v.t
}

// LogValuer returns v's value as a LogValuer. It panics if v is not a LogValuer.
func (v Value) LogValuer() LogValuer {
	return v.any.(LogValuer)
}

// Group returns v's value as a []Attr. It panics if v's Kind is not KindGroup.
func (v Value) Group() []Attr {
	if v.kind != KindGroup {
		panic("Group: bad kind")
	}
	return v.group
}

// Equal reports whether v and w represent the same Go value.
func (v Value) Equal(w Value) bool {
	if v.kind != w.kind {
		return false
	}
	switch v.kind {
	case KindInt64, KindUint64, KindBool, KindDuration:
		return v.num == w.num
	case KindString:
		return v.str == w.str
	case KindFloat64:
		return v.float == w.float
	case KindTime:
		return v.t.Equal(w.t)
	case KindAny, KindLogValuer:
		return v.any == w.any
	case KindGroup:
		if len(v.group) != len(w.group) {
			return false
		}
		for i := range v.group {
			if !v.group[i].Equal(w.group[i]) {
				return false
			}
		}
		return true
	default:
		panic("bad kind: " + v.kind.String())
	}
}

func (v Value) isEmptyGroup() bool {
	return v.kind == KindGroup && len(v.group) == 0
}

func (v Value) append(dst []byte) []byte {
	switch v.kind {
	case KindString:
		return append(dst, v.str...)
	case KindInt64:
		return strconv.AppendInt(dst, int64(v.num), 10)
	case KindUint64:
		return strconv.AppendUint(dst, v.num, 10)
	case KindFloat64:
		return strconv.AppendFloat(dst, v.float, 'g', -1, 64)
	case KindBool:
		return append(dst, strconv.FormatBool(v.num == 1)...)
	case KindDuration:
		return append(dst, time.Duration(int64(v.num)).String()...)
	case KindTime:
		return append(dst, v.t.String()...)
	case KindGroup:
		dst = append(dst, '[')
		for i, a := range v.group {
			if i > 0 {
				dst = append(dst, ' ')
			}
			dst = append(dst, a.String()...)
		}
		return append(dst, ']')
	case KindAny, KindLogValuer:
		return fmt.Append(dst, v.any)
	default:
		panic("bad kind: " + v.kind.String())
	}
}

// A LogValuer is any Go value that can convert itself into a Value for
// logging.
type LogValuer interface {
	LogValue() Value
}

const maxLogValues = 100

// Resolve repeatedly calls LogValue on v while it implements LogValuer, and
// returns the result. A LogValue panic resolves to an error value.
func (v Value) Resolve() (rv Value) {
	defer func() {
		if r := recover(); r != nil {
			rv = AnyValue(fmt.Errorf("LogValue panicked: %v", r))
		}
	}()
	for i := 0; i < maxLogValues; i++ {
		if v.kind != KindLogValuer {
			return v
		}
		v = v.LogValuer().LogValue()
	}
	return AnyValue(fmt.Errorf("LogValue called too many times on Value of type %T", v.Any()))
}

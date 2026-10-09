package slog

import "github.com/eugenioenko/goalchemy/std/time"

// An Attr is a key-value pair.
type Attr struct {
	Key   string
	Value Value
}

// String returns an Attr for a string value.
func String(key, value string) Attr { return Attr{key, StringValue(value)} }

// Int64 returns an Attr for an int64.
func Int64(key string, value int64) Attr { return Attr{key, Int64Value(value)} }

// Int returns an Attr for an int, stored as an int64.
func Int(key string, value int) Attr { return Int64(key, int64(value)) }

// Uint64 returns an Attr for a uint64.
func Uint64(key string, v uint64) Attr { return Attr{key, Uint64Value(v)} }

// Float64 returns an Attr for a float64.
func Float64(key string, v float64) Attr { return Attr{key, Float64Value(v)} }

// Bool returns an Attr for a bool.
func Bool(key string, v bool) Attr { return Attr{key, BoolValue(v)} }

// Time returns an Attr for a time.Time.
func Time(key string, v time.Time) Attr { return Attr{key, TimeValue(v)} }

// Duration returns an Attr for a time.Duration.
func Duration(key string, v time.Duration) Attr { return Attr{key, DurationValue(v)} }

// Group returns an Attr for a group of attributes built from alternating
// keys and values, as in Logger.Log.
func Group(key string, args ...any) Attr {
	return Attr{key, GroupValue(argsToAttrSlice(args)...)}
}

// GroupAttrs returns an Attr for a group of attrs.
func GroupAttrs(key string, attrs ...Attr) Attr {
	return Attr{key, GroupValue(attrs...)}
}

func argsToAttrSlice(args []any) []Attr {
	var attrs []Attr
	for len(args) > 0 {
		var attr Attr
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}

// Any returns an Attr for value, as AnyValue converts it.
func Any(key string, value any) Attr { return Attr{key, AnyValue(value)} }

// Equal reports whether a and b have equal keys and values.
func (a Attr) Equal(b Attr) bool {
	return a.Key == b.Key && a.Value.Equal(b.Value)
}

// String returns key=value.
func (a Attr) String() string {
	return a.Key + "=" + a.Value.String()
}

func (a Attr) isEmpty() bool {
	return a.Key == "" && a.Value.kind == KindAny && a.Value.any == nil
}

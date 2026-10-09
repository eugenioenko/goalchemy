package slog

import "github.com/eugenioenko/goalchemy/std/time"

// A Record holds information about a log event.
type Record struct {
	// Time is when the record was created; the zero time is omitted.
	Time time.Time
	// Message is the log message.
	Message string
	// Level is the level of the event.
	Level Level
	// PC is always zero: source locations are not recorded.
	PC int

	attrs []Attr
}

// NewRecord creates a Record from the given arguments. Use Record.AddAttrs
// to add attributes. pc is kept for source compatibility and is ignored.
func NewRecord(t time.Time, level Level, msg string, pc int) Record {
	return Record{Time: t, Message: msg, Level: level}
}

// Clone returns a copy of the record whose attributes are not shared.
func (r Record) Clone() Record {
	r.attrs = append([]Attr(nil), r.attrs...)
	return r
}

// NumAttrs returns the number of attributes in the Record.
func (r Record) NumAttrs() int { return len(r.attrs) }

// Attrs calls f on each Attr in the Record. Iteration stops if f returns
// false.
func (r Record) Attrs(f func(Attr) bool) {
	for _, a := range r.attrs {
		if !f(a) {
			return
		}
	}
}

func (r *Record) grow(n int) {
	attrs := make([]Attr, len(r.attrs), len(r.attrs)+n)
	copy(attrs, r.attrs)
	r.attrs = attrs
}

// AddAttrs appends the given Attrs to the Record's list, omitting empty
// groups.
func (r *Record) AddAttrs(attrs ...Attr) {
	r.grow(len(attrs))
	for _, a := range attrs {
		if !a.Value.isEmptyGroup() {
			r.attrs = append(r.attrs, a)
		}
	}
}

// Add converts args to Attrs as Logger.Log does and appends them to the
// Record's list, omitting empty groups.
func (r *Record) Add(args ...any) {
	r.grow(len(args))
	for len(args) > 0 {
		var a Attr
		a, args = argsToAttr(args)
		if !a.Value.isEmptyGroup() {
			r.attrs = append(r.attrs, a)
		}
	}
}

const badKey = "!BADKEY"

func argsToAttr(args []any) (Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return String(badKey, x), nil
		}
		return Any(x, args[1]), args[2:]
	case Attr:
		return x, args[1:]
	default:
		return Any(badKey, x), args[1:]
	}
}

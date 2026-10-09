package fmt_test

import (
	stderrors "errors"
	stdfmt "fmt"
	"math"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
)

type celsius float64

func (c celsius) String() string { return stdfmt.Sprintf("%.1f°C", float64(c)) }

type code int

func (c *code) Error() string { return "code " + stdfmt.Sprint(int(*c)) }

type panicky struct{}

func (panicky) String() string { panic("boom") }

type goString struct{}

func (goString) GoString() string { return "goString{}" }
func (goString) String() string   { return "plain" }

type stdFormatted struct{}

func (stdFormatted) Format(f stdfmt.State, verb rune) {
	w, wok := f.Width()
	p, pok := f.Precision()
	stdfmt.Fprintf(f, "[%c %d %v %d %v %v%v%v%v%v %s]", verb, w, wok, p, pok,
		f.Flag('-'), f.Flag('+'), f.Flag('#'), f.Flag(' '), f.Flag('0'), stdfmt.FormatString(f, verb))
}

type formatted struct{}

func (formatted) Format(f fmt.State, verb rune) {
	w, wok := f.Width()
	p, pok := f.Precision()
	f.Write([]byte(fmt.Sprintf("[%c %d %v %d %v %v%v%v%v%v %s]", verb, w, wok, p, pok,
		f.Flag('-'), f.Flag('+'), f.Flag('#'), f.Flag(' '), f.Flag('0'), fmt.FormatString(f, verb))))
}

var basics = []any{
	nil, true, false, 0, 1, -1, 42, -42, 255, 65, 0x1F600, 0x110000, math.MaxInt64, math.MinInt64,
	int8(-128), int16(1234), int32(-5), int64(1 << 40), uint(7), uint8(200), uint16(65535), uint32(1 << 31),
	uint64(math.MaxUint64), 0.0, math.Copysign(0, -1), 1.0, -1.5, 3.14159, 1e6, 1e21, 1e-7, 123456789.0,
	math.Inf(1), math.Inf(-1), math.NaN(), float32(0.1), float32(-2.5), float32(1e30), 5e-324,
	"", "hello", "日本語", "a\"b\\c\n", "\xff\xfe", "tab\there", "`back`", []byte("bytes"), []byte{}, []byte(nil),
	[]byte{0, 1, 0xff},
}

var verbs = "vdbcoOqxXUeEfFgGstTw%z"

var shapes = []string{"", "+", "-", "#", " ", "0", "+#", "-0", " 0", "#0", "6", "-6", "06", "+08", ".0", ".3", "8.3", "-8.3", "+.2", "#.0", "# x"}

func TestSprintfMatchesStd(t *testing.T) {
	for _, arg := range basics {
		for _, shape := range shapes {
			for _, verb := range verbs {
				format := "%" + shape + string(verb)
				if g, w := fmt.Sprintf(format, arg), stdfmt.Sprintf(format, arg); g != w {
					t.Errorf("Sprintf(%q, %#v) = %q want %q", format, arg, g, w)
				}
			}
		}
	}
}

func TestMethodsMatchStd(t *testing.T) {
	c := code(7)
	var nilCode *code
	args := []any{celsius(21.5), &c, nilCode, stderrors.New("boom"), panicky{}}
	for _, arg := range args {
		for _, shape := range shapes {
			for _, verb := range "vsqxX" {
				if verb == 'v' && strings.Contains(shape, "#") {
					continue
				}
				format := "%" + shape + string(verb)
				want := stdfmt.Sprintf(format, arg)
				if arg == args[3] {
					arg = errors.New("boom")
				}
				if g := fmt.Sprintf(format, arg); g != want {
					t.Errorf("Sprintf(%q, %T) = %q want %q", format, arg, g, want)
				}
			}
		}
	}
	for _, format := range []string{"%#v", "%v", "%s", "%+v", "%#q"} {
		if g, w := fmt.Sprintf(format, goString{}), stdfmt.Sprintf(format, goString{}); g != w {
			t.Errorf("Sprintf(%q, goString{}) = %q want %q", format, g, w)
		}
	}
	for _, shape := range shapes {
		for _, verb := range "vsdqx" {
			format := "%" + shape + string(verb)
			if g, w := fmt.Sprintf(format, formatted{}), stdfmt.Sprintf(format, stdFormatted{}); g != w {
				t.Errorf("Sprintf(%q, Formatter) = %q want %q", format, g, w)
			}
		}
	}
}

func TestFormatsMatchStd(t *testing.T) {
	cases := []struct {
		format string
		args   []any
	}{
		{"%d %d", []any{1}},
		{"%d", []any{1, 2, "x", nil}},
		{"%[2]d %[1]d", []any{1, 2}},
		{"%[3]d", []any{1, 2}},
		{"%[0]d", []any{1}},
		{"%[x]d", []any{1}},
		{"%[2]*[1]d", []any{12, 5}},
		{"%*d|%-*d", []any{5, 1, 5, 2}},
		{"%*d", []any{"x", 1}},
		{"%.*f", []any{2, math.Pi}},
		{"%.*f", []any{-2, math.Pi}},
		{"%*d", []any{uint(3), 1}},
		{"%*d", []any{10000000, 1}},
		{"%", nil},
		{"%-", []any{1}},
		{"%!", []any{1}},
		{"100%%", nil},
		{"%5%", nil},
		{"%v %v", []any{nil, []byte(nil)}},
		{"%s", []any{nil}},
		{"%d", []any{nil}},
		{"%.2s|%.0s|%5.1s", []any{"日本語", "x", "ab"}},
		{"%x|% x|%#x|% #x|%X", []any{"hi", "hi", "hi", "hi", []byte("hi")}},
		{"%10.4x|%-10x", []any{"hello", []byte("hi")}},
		{"%c|%c|%q|%+q|%#q|%U|%#U|%#14U", []any{'x', 0x1F600, 'x', '日', '\'', 0x1F600, 'x', 'x'}},
		{"%q|%+q|%#q|%#+q", []any{"日本\n", "日本", "a`b", "ok"}},
		{"%t|%v|%5t", []any{true, false, true}},
		{"%e|%E|%.3g|%#g|%#.3x|%x|%b|%#v", []any{1234.5678, 1e-10, 1234.5678, 1.0, 1.0, 3.5, 1.0, 2.0}},
		{"%08.3f|%-8.3f|%+.0e|% .2f|%+08d|% d|%x|%X|%#o|%O|%#b", []any{-3.14159, 2.5, 12345.0, 2.0, -42, 42, -255, 255, 8, 8, 5}},
		{"%v|%v|%v|%v", []any{float32(1) / 3, 1.0 / 3, 1e20, 1e21}},
		{"%6.2v|%.3v", []any{math.Pi, "abcdef"}},
		{"%#v|%#v|%#v|%#v|%#v", []any{"s", 42, uint(42), []byte{1, 2}, []byte(nil)}},
		{"%c|%o|%t", []any{[]byte("hi"), []byte{8, 9}, []byte{1}}},
		{"%T|%T|%T|%T", []any{1, "s", []byte{}, nil}},
		{"%s %d", []any{"a", "b"}},
		{"%w", []any{1}},
		{"%v", []any{[]byte("x")}},
		{"%010s|%-010s|%010q", []any{"x", "x", "x"}},
		{"%.1f|%.1f|%.0f|%.0f", []any{0.25, 0.35, 0.5, 1.5}},
	}
	for _, c := range cases {
		if g, w := fmt.Sprintf(c.format, c.args...), stdfmt.Sprintf(c.format, c.args...); g != w {
			t.Errorf("Sprintf(%q, %v) = %q want %q", c.format, c.args, g, w)
		}
	}
	prints := [][]any{
		nil, {1, 2}, {"a", "b"}, {1, "a", 2}, {"a", 1, "b"}, {nil, nil}, {1.5, true, []byte("x")}, {"x", nil, 3},
	}
	for _, a := range prints {
		if g, w := fmt.Sprint(a...), stdfmt.Sprint(a...); g != w {
			t.Errorf("Sprint(%v) = %q want %q", a, g, w)
		}
		if g, w := fmt.Sprintln(a...), stdfmt.Sprintln(a...); g != w {
			t.Errorf("Sprintln(%v) = %q want %q", a, g, w)
		}
		if g, w := string(fmt.Append([]byte("p:"), a...)), string(stdfmt.Append([]byte("p:"), a...)); g != w {
			t.Errorf("Append(%v) = %q want %q", a, g, w)
		}
		if g, w := string(fmt.Appendln([]byte("p:"), a...)), string(stdfmt.Appendln([]byte("p:"), a...)); g != w {
			t.Errorf("Appendln(%v) = %q want %q", a, g, w)
		}
	}
	if g, w := string(fmt.Appendf([]byte("p:"), "%d-%s", 1, "x")), string(stdfmt.Appendf([]byte("p:"), "%d-%s", 1, "x")); g != w {
		t.Errorf("Appendf = %q want %q", g, w)
	}
}

func TestUnsupportedOperands(t *testing.T) {
	type point struct{ x, y int }
	for _, c := range []struct {
		format string
		arg    any
		want   string
	}{
		{"%v", point{1, 2}, "%!v(unsupported)"},
		{"%d", celsius(1), "%!d(unsupported)"},
		{"%T", celsius(1), "%!T(unsupported)"},
		{"%p", []byte{}, "%!p(unsupported)"},
		{"%d|%v", celsius(2), "%!d(unsupported)|%!v(MISSING)"},
	} {
		if g := fmt.Sprintf(c.format, c.arg); g != c.want {
			t.Errorf("Sprintf(%q, %v) = %q want %q", c.format, c.arg, g, c.want)
		}
	}
	if g, w := fmt.Sprintf("%d", 1, celsius(3)), "1%!(EXTRA ?=3.0°C)"; g != w {
		t.Errorf("extra = %q want %q", g, w)
	}
}

func TestErrorfMatchesStd(t *testing.T) {
	base := errors.New("base")
	stdBase := stderrors.New("base")
	other := errors.New("other")
	stdOther := stderrors.New("other")
	cases := []struct {
		format string
		args   func(base, other error) []any
	}{
		{"plain", func(b, o error) []any { return nil }},
		{"100%%", func(b, o error) []any { return nil }},
		{"wrap: %w", func(b, o error) []any { return []any{b} }},
		{"wrap: %v", func(b, o error) []any { return []any{b} }},
		{"%w and %w", func(b, o error) []any { return []any{b, o} }},
		{"%[2]w then %[1]w and %[2]w", func(b, o error) []any { return []any{b, o} }},
		{"%w", func(b, o error) []any { return []any{"not an error"} }},
		{"%w %d", func(b, o error) []any { return []any{nil, 3} }},
		{"%s: %w", func(b, o error) []any { return []any{"ctx", b} }},
	}
	for _, c := range cases {
		got := fmt.Errorf(c.format, c.args(base, other)...)
		want := stdfmt.Errorf(c.format, c.args(stdBase, stdOther)...)
		if got.Error() != want.Error() {
			t.Errorf("Errorf(%q) = %q want %q", c.format, got.Error(), want.Error())
		}
		if g, w := errors.Is(got, base), stderrors.Is(want, stdBase); g != w {
			t.Errorf("Errorf(%q) Is base = %v want %v", c.format, g, w)
		}
		if g, w := errors.Is(got, other), stderrors.Is(want, stdOther); g != w {
			t.Errorf("Errorf(%q) Is other = %v want %v", c.format, g, w)
		}
		if g, w := errors.Unwrap(got) == base, stderrors.Unwrap(want) == stdBase; g != w {
			t.Errorf("Errorf(%q) Unwrap = %v want %v", c.format, g, w)
		}
		_, gm := got.(interface{ Unwrap() []error })
		_, wm := want.(interface{ Unwrap() []error })
		if gm != wm {
			t.Errorf("Errorf(%q) multi = %v want %v", c.format, gm, wm)
		}
	}
	if got := fmt.Errorf("%w %w", base, base).(interface{ Unwrap() []error }).Unwrap(); len(got) != 2 {
		t.Errorf("duplicate %%w operands = %d", len(got))
	}
	if got := fmt.Errorf("%[1]w %[1]w", base).(interface{ Unwrap() []error }).Unwrap(); len(got) != 1 {
		t.Errorf("reordered duplicate %%w operands = %d", len(got))
	}
	if !strings.Contains(fmt.Sprintf("%v", fmt.Errorf("x: %w", base)), "x: base") {
		t.Error("Errorf value formatting")
	}
}

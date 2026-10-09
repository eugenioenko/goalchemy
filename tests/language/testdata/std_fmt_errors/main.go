package main

import (
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/time"
)

var ErrNotFound = errors.New("not found")

type KASError struct {
	URL    string
	Status int
}

func (e *KASError) Error() string { return fmt.Sprintf("kas %s: status %d", e.URL, e.Status) }

type codeError struct{ code int }

func (e codeError) Error() string { return fmt.Sprint("code ", e.code) }

type level int

func (l level) String() string { return [...]string{"low", "mid", "high"}[l] }

type celsius float64

func (c *celsius) String() string { return fmt.Sprintf("%.1f°C", float64(*c)) }

type quiet struct{}

func (quiet) String() string { panic("noisy") }

type tagged struct{ name string }

func (t tagged) GoString() string { return "tagged(" + t.name + ")" }
func (t tagged) String() string   { return t.name }

type box struct{ n int }

func (b box) Format(f fmt.State, verb rune) {
	w, ok := f.Width()
	f.Write([]byte(fmt.Sprintf("box<%c,%d,%v,%v,%s>", verb, w, ok, f.Flag('+'), fmt.FormatString(f, verb))))
}

type asKAS struct{}

func (asKAS) Error() string { return "asKAS" }

func (asKAS) As(target any) bool {
	if p, ok := target.(**KASError); ok {
		*p = &KASError{URL: "via As", Status: 1}
		return true
	}
	return false
}

type multi struct{ errs []error }

func (m multi) Error() string   { return "multi" }
func (m multi) Unwrap() []error { return m.errs }

type sliceErr []int

func (s sliceErr) Error() string { return "slice" }

func rewrap(url string, err error) error {
	return fmt.Errorf("rewrap %s: %w", url, err)
}

func formats() {
	println(fmt.Sprintf("%d segments, %x, %X, %o, %O, %b, %c, %q, %U, %#U", 12, 255, 255, 8, 8, 5, 'G', 'x', 0x1F600, 'é'))
	println(fmt.Sprintf("[%5d|%-5d|%05d|%+d|% d|%x|%#x|%#o|%#b]", 42, 42, -42, 42, 42, -255, 255, 8, 5))
	println(fmt.Sprintf("[%s|%10s|%-10s|%.2s|%q|%+q|%#q|%x|% X|%#x]", "go", "go", "go", "日本語", "a\"b\n", "日本", "a`b", "hi", "hi", "hi"))
	println(fmt.Sprintf("[%v|%v|%v|%v|%v|%v]", 3.14159, 1e21, 1e20, 1e-7, float32(0.1), 100.0))
	println(fmt.Sprintf("[%f|%.2f|%8.3f|%-8.1f|%08.2f|%+.1e|%E|%g|%G|%.3g|%#g]", 3.14159, 2.675, -1.5, 2.25, -3.7, 12345.678, 1e-10, 1e6, 1e-5, 1234.5678, 2.0))
	println(fmt.Sprintf("[%x|%X|%b|%.3x|%v|%v|%v|%5.1f|%v]", 1.0, -2.5, 8.0, 3.14159, 0.0, -0.0*1, 5e-324, 9.96, 1.7976931348623157e308))
	nan, inf := 0.0, 1.0
	nan, inf = nan/nan, inf/0
	println(fmt.Sprintf("[%v|%v|%v|%5.2f|%+v|%08v]", nan, inf, -inf, nan, inf, -inf))
	println(fmt.Sprintf("[%t|%v|%5t|%v|%s|%d|%x|%v|%v]", true, false, true, []byte("hi"), []byte("hi"), []byte{1, 2}, []byte{0xca, 0xfe}, nil, []byte(nil)))
	println(fmt.Sprintf("[%#v|%#v|%#v|%#v|%T|%T|%T|%T]", "s", 42, uint8(42), []byte{1}, 1, "s", 2.5, []byte{}))
	println(fmt.Sprintf("%[2]d %[1]d %d|%*d|%-*d|%.*f|%[3]*.[2]*[1]f", 1, 2, 3, 5, 7, 4, 8, 2, 3.14159))
	println(fmt.Sprintf("%d %s|%!|%z|%d|%[5]d|100%%", "x", 3, 1.5, 1))
	println(fmt.Sprintf("%d", 1, "extra", 2.5, nil))
	println(fmt.Sprint("a", 1, 2, "b", "c", 3.5, true, nil, []byte("z")))
	print(fmt.Sprintln("a", 1, 2, "b", nil))
	println(string(fmt.Appendf([]byte("pre:"), "%03d", 7)), string(fmt.Append(nil, 1, 2)), string(fmt.Appendln(nil, "x")))
}

func methods() {
	c := celsius(21.5)
	var nilC *celsius
	var lvl fmt.Stringer = level(2)
	println(fmt.Sprintf("[%v|%s|%q|%x|%10v|%-6s|%d]", level(1), level(0), level(2), level(1), lvl, &c, level(1)))
	println(fmt.Sprintf("[%v|%s|%v]", nilC, quiet{}, &KASError{"k", 503}))
	println(fmt.Sprintf("[%v|%#v|%s|%+8v|%-3x]", box{1}, box{2}, box{3}, box{4}, box{5}))
	println(fmt.Sprintf("[%v|%#v|%s]", tagged{"t"}, tagged{"t"}, tagged{"t"}))
	var anything any = struct{ a, b int }{1, 2}
	var p any = &struct{}{}
	println(fmt.Sprintf("[%v|%d|%T|%p|%v]", anything, codeError{1}, codeError{2}, p, map[string]int(nil) == nil))
	println(fmt.Sprintf("%v %d", codeError{3}, codeError{4}), fmt.Sprint(codeError{5}, errors.New("e")))
}

func wrapping() {
	err := rewrap("https://kas", &KASError{"https://kas", 403})
	println(err.Error())
	var kerr *KASError
	println(errors.As(err, &kerr), kerr.Status, kerr.URL)
	var ce codeError
	println(errors.As(err, &ce), ce.code)
	println(errors.As(fmt.Errorf("x: %w", codeError{7}), &ce), ce.code)
	var unwrapper interface{ Unwrap() error }
	println(errors.As(err, &unwrapper), unwrapper.Unwrap().Error())
	var target error
	println(errors.As(err, &target), target.Error())

	nf := fmt.Errorf("lookup %q: %w", "key", ErrNotFound)
	println(nf.Error(), errors.Is(nf, ErrNotFound), errors.Is(nf, errors.New("not found")), errors.Unwrap(nf) == ErrNotFound)

	two := fmt.Errorf("both: %w and %w", ErrNotFound, codeError{9})
	println(two.Error(), errors.Is(two, ErrNotFound), errors.Is(two, codeError{9}), errors.Is(two, codeError{8}), errors.Unwrap(two) == nil)
	println(errors.As(two, &ce), ce.code)
	reordered := fmt.Errorf("%[2]w %[1]w %[2]w", ErrNotFound, codeError{10})
	println(reordered.Error(), len(reordered.(interface{ Unwrap() []error }).Unwrap()))
	println(fmt.Errorf("%w", "not an error").Error(), fmt.Errorf("plain").Error(), fmt.Errorf("%v", ErrNotFound).Error())
	println(errors.Unwrap(fmt.Errorf("%w", nil)) == nil, fmt.Errorf("%w", nil).Error())

	joined := errors.Join(nil, ErrNotFound, nil, &KASError{"j", 500})
	println(joined.Error(), errors.Is(joined, ErrNotFound), errors.As(joined, &kerr), kerr.URL, errors.Join(nil, nil) == nil)
	deep := multi{[]error{nil, fmt.Errorf("level1: %w", multi{[]error{codeError{11}}})}}
	println(errors.Is(deep, codeError{11}), errors.As(deep, &ce), ce.code)

	println(errors.As(asKAS{}, &kerr), kerr.URL, errors.As(fmt.Errorf("w: %w", asKAS{}), &kerr), kerr.Status)
	println(errors.Is(sliceErr{1}, sliceErr{1}), errors.Is(fmt.Errorf("%w", sliceErr{1}), sliceErr{1}))
	println(errors.As(nil, &kerr), errors.Is(nil, nil), errors.Is(nil, ErrNotFound), errors.Unwrap(nil) == nil)
	println(errors.ErrUnsupported.Error())

	defer func() {
		println("recovered:", fmt.Sprint(recover()))
	}()
	var nilTarget **KASError
	errors.As(ErrNotFound, nilTarget)
}

func main() {
	formats()
	methods()
	wrapping()
	println(fmt.Sprintf("%v|%s", time.Duration(1500)*time.Millisecond, time.Unix(0, 0)))
}

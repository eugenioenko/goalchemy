package fmt

import (
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/sort"
	"github.com/eugenioenko/goalchemy/std/strings"
)

// Errorf formats according to a format specifier and returns the string as
// a value that satisfies error.
//
// If the format specifier includes a %w verb with an error operand, the
// returned error will implement an Unwrap method returning the operand. If
// there is more than one %w verb, the returned error will implement an
// Unwrap method returning a []error containing all the %w operands in the
// order they appear in the arguments. It is invalid to supply the %w verb
// with an operand that does not implement the error interface. The %w verb
// is otherwise a synonym for %v.
func Errorf(format string, a ...any) error {
	if len(a) == 0 && strings.IndexByte(format, '%') == -1 {
		return errors.New(format)
	}
	p := newPrinter()
	p.wrapErrs = true
	p.doPrintf(format, a)
	s := string(p.buf)
	switch len(p.wrappedErrs) {
	case 0:
		return errors.New(s)
	case 1:
		w := &wrapError{msg: s}
		w.err, _ = a[p.wrappedErrs[0]].(error)
		return w
	}
	if p.reordered {
		sort.Ints(p.wrappedErrs)
	}
	var errs []error
	for i, argNum := range p.wrappedErrs {
		if i > 0 && p.wrappedErrs[i-1] == argNum {
			continue
		}
		if e, ok := a[argNum].(error); ok {
			errs = append(errs, e)
		}
	}
	return &wrapErrors{s, errs}
}

type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

type wrapErrors struct {
	msg  string
	errs []error
}

func (e *wrapErrors) Error() string {
	return e.msg
}

func (e *wrapErrors) Unwrap() []error {
	return e.errs
}

// Package errors implements functions to manipulate errors: creation,
// wrapping inspection with Is and As, and joining. As requires a target whose
// static type is a pointer to an interface type or to a type implementing
// error; the compiler checks it at the call site instead of using reflection.
package errors

import liberrors "github.com/eugenioenko/goalchemy/lib/errors"

// New returns an error that formats as the given text. Each call to New
// returns a distinct error value even if the text is identical.
func New(text string) error { return liberrors.New(text) }

// ErrUnsupported indicates that a requested operation cannot be performed,
// because it is unsupported.
var ErrUnsupported = New("unsupported operation")

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error. Otherwise, Unwrap returns
// nil. It does not unwrap errors returned by Join.
func Unwrap(err error) error {
	u, ok := err.(interface{ Unwrap() error })
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// Is reports whether any error in err's tree matches target. The tree is err
// followed by a depth-first traversal of the errors returned by its
// Unwrap() error or Unwrap() []error methods. An error matches target if it
// is equal to target or has a method Is(error) bool returning true.
func Is(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	return is(err, target)
}

func is(err, target error) bool {
	for {
		if equal(err, target) {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if is(err, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// equal compares like errors.Is: a target of a non-comparable dynamic type
// never matches by equality. Interface comparison panics only when both
// dynamic types are the same non-comparable type.
func equal(err, target error) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return err == target
}

// As finds the first error in err's tree that matches target, and if one is
// found, sets target to that error value and returns true. Otherwise, it
// returns false. An error matches if its concrete value is assignable to the
// value target points to, or if it has a method As(any) bool returning true.
//
// The compiler expands each call for the static type of target, which must
// be a pointer to an interface type or to a type implementing error. As
// panics if target is a nil pointer and err is not nil.
func As(err error, target any) bool { return liberrors.As(err, target) }

func asTarget(err error, target any, targetNil bool, match func(error, any) bool) bool {
	if err == nil {
		return false
	}
	if targetNil {
		panic("errors: target must be a non-nil pointer")
	}
	return as(err, target, match)
}

func as(err error, target any, match func(error, any) bool) bool {
	for {
		if match(err, target) {
			return true
		}
		if x, ok := err.(interface{ As(any) bool }); ok && x.As(target) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				if as(err, target, match) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// Join returns an error that wraps the given errors. Any nil error values
// are discarded. Join returns nil if every value in errs is nil. The error
// formats as the concatenation of the strings obtained by calling the Error
// method of each element of errs, with a newline between each string.
func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{errs: make([]error, 0, n)}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	if len(e.errs) == 1 {
		return e.errs[0].Error()
	}
	b := []byte(e.errs[0].Error())
	for _, err := range e.errs[1:] {
		b = append(b, '\n')
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (e *joinError) Unwrap() []error {
	return e.errs
}

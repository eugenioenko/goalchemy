package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/eugenioenko/goalchemy/std/errors"
)

type kasError struct{ url string }

func (e *kasError) Error() string { return "kas " + e.url }

type valueError struct{ code int }

func (e valueError) Error() string { return "value" }

type sliceError []string

func (e sliceError) Error() string { return "slice" }

type wrapper struct {
	msg string
	err error
}

func (w *wrapper) Error() string { return w.msg }
func (w *wrapper) Unwrap() error { return w.err }

type multi struct{ errs []error }

func (m multi) Error() string   { return "multi" }
func (m multi) Unwrap() []error { return m.errs }

type matchAll struct{}

func (matchAll) Error() string        { return "all" }
func (matchAll) Is(target error) bool { return target != nil && target.Error() == "sentinel" }

type asKAS struct{}

func (asKAS) Error() string { return "asKAS" }
func (asKAS) As(target any) bool {
	if p, ok := target.(**kasError); ok {
		*p = &kasError{url: "from As"}
		return true
	}
	return false
}

func TestIsMatchesStd(t *testing.T) {
	sentinel := errors.New("sentinel")
	kas := &kasError{"k"}
	trees := []error{
		nil, sentinel, kas, valueError{1}, sliceError{"a"}, matchAll{},
		&wrapper{"w", sentinel}, &wrapper{"w", &wrapper{"w2", kas}}, &wrapper{"w", nil},
		multi{[]error{kas, &wrapper{"w", sentinel}}}, multi{nil}, multi{[]error{nil, valueError{2}}},
		errors.Join(kas, nil, sentinel), errors.Join(), errors.Join(nil),
	}
	targets := []error{nil, sentinel, kas, valueError{1}, valueError{2}, sliceError{"a"}, errors.New("sentinel")}
	for _, err := range trees {
		for _, target := range targets {
			if g, w := errors.Is(err, target), stderrors.Is(err, target); g != w {
				t.Errorf("Is(%v, %v) = %v want %v", err, target, g, w)
			}
		}
		if g, w := errors.Unwrap(err), stderrors.Unwrap(err); g != w {
			t.Errorf("Unwrap(%v) = %v want %v", err, g, w)
		}
	}
}

func TestAsMatchesStd(t *testing.T) {
	kas := &kasError{"k"}
	trees := []error{
		nil, kas, valueError{3}, &wrapper{"w", kas}, multi{[]error{nil, valueError{4}, kas}},
		errors.Join(errors.New("x"), &wrapper{"w", valueError{5}}), asKAS{}, &wrapper{"w", nil},
	}
	for _, err := range trees {
		var gk, wk *kasError
		if g, w := errors.As(err, &gk), stderrors.As(err, &wk); g != w || (gk == nil) != (wk == nil) || gk != nil && gk.url != wk.url {
			t.Errorf("As(%v, *kasError) = %v %v want %v %v", err, g, gk, w, wk)
		}
		var gv, wv valueError
		if g, w := errors.As(err, &gv), stderrors.As(err, &wv); g != w || gv != wv {
			t.Errorf("As(%v, valueError) = %v %v want %v %v", err, g, gv, w, wv)
		}
		var gi, wi interface{ Unwrap() error }
		if g, w := errors.As(err, &gi), stderrors.As(err, &wi); g != w || gi != wi {
			t.Errorf("As(%v, unwrapper) = %v want %v", err, g, w)
		}
	}
}

func TestAsNilTargetPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != "errors: target must be a non-nil pointer" {
			t.Errorf("panic %v", r)
		}
	}()
	var p **kasError
	errors.As(errors.New("x"), p)
}

func TestJoinMatchesStd(t *testing.T) {
	a, b := errors.New("a"), errors.New("b")
	for _, errs := range [][]error{nil, {nil}, {a}, {a, nil, b}, {nil, b}} {
		g, w := errors.Join(errs...), stderrors.Join(errs...)
		if (g == nil) != (w == nil) {
			t.Fatalf("Join(%v) = %v want %v", errs, g, w)
		}
		if g != nil && g.Error() != w.Error() {
			t.Errorf("Join(%v) = %q want %q", errs, g.Error(), w.Error())
		}
	}
	if errors.ErrUnsupported.Error() != stderrors.ErrUnsupported.Error() {
		t.Error("ErrUnsupported text")
	}
}

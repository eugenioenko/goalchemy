package main

import "errors"

var ErrNotFound = errors.New("not found")

type ValidationError struct {
	Field string
}

func (v ValidationError) Error() string { return "invalid " + v.Field }

type Wrapped struct {
	msg string
	err error
}

func (w *Wrapped) Error() string { return w.msg + ": " + w.err.Error() }
func (w *Wrapped) Unwrap() error { return w.err }

func lookup(k string) (int, error) {
	if k == "" {
		return 0, ValidationError{"key"}
	}
	if k != "a" {
		return 0, &Wrapped{"lookup " + k, ErrNotFound}
	}
	return 1, nil
}

func main() {
	for _, k := range []string{"a", "b", ""} {
		v, err := lookup(k)
		if err != nil {
			println(k, "error:", err.Error(), errors.Is(err, ErrNotFound))
			if ve, ok := err.(ValidationError); ok {
				println("field", ve.Field)
			}
			continue
		}
		println(k, v)
	}
	e1 := errors.New("same")
	e2 := errors.New("same")
	println(e1 == e2, e1.Error() == e2.Error(), errors.Is(e1, e1))
	w := &Wrapped{"outer", &Wrapped{"inner", ErrNotFound}}
	println(errors.Unwrap(w) != nil, errors.Unwrap(ErrNotFound) == nil, errors.Is(w, ErrNotFound))
	var err error = ValidationError{"x"}
	println(err == ValidationError{"x"}, err == ValidationError{"y"})
	panic(&Wrapped{"fatal", ErrNotFound})
}

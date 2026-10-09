package main

// goalchemy:reject GCS006

import "github.com/eugenioenko/goalchemy/std/errors"

type codeError struct{ code int }

func (e *codeError) Error() string { return "code" }

func main() {
	var target codeError
	println(errors.As(errors.New("x"), &target))
}

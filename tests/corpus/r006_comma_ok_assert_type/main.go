package main

import "github.com/eugenioenko/goalchemy/lib/errors"

func main() {
	var v any = errors.New("e")
	if e, ok := v.(error); ok {
		println(e.Error())
	}
	n, ok := v.(int)
	println(n, ok)
}

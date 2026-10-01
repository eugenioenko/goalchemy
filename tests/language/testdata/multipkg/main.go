package main

import "github.com/eugenioenko/goalchemy/tests/language/testdata/multipkg/inner"

var top = inner.Value * 2

func init() { println("main init", top, inner.Count()) }

func main() {
	c := inner.NewCounter(3)
	c.Add(4)
	println(c.Total(), inner.Value, inner.Describe(c))
	var s inner.Shape = inner.Square{Side: 2}
	println(s.Area())
}

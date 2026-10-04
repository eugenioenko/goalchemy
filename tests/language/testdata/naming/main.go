package main

import (
	left "github.com/eugenioenko/goalchemy/tests/language/testdata/naming/left"
	right "github.com/eugenioenko/goalchemy/tests/language/testdata/naming/right"
)

type PolicyBinding struct {
	Algorithm    string
	Hash         string
	LegacyString bool
}
type Keywords struct {
	constructor int
	clone       int
	class       int
	match       int
	namespace   int
	def         int
}
type Δ struct{ Value int }
type _u0394_ struct{ Value int }
type Measured interface{ Value() int }

var class = 11
var Class = 17

func evaluate(binding PolicyBinding, class int) (result int) {
	result = class + len(binding.Algorithm)
	{
		class := 3
		result += class
	}
	snapshot := binding
	snapshot.Hash = "copy"
	println(binding.Hash, snapshot.Hash, binding == snapshot)
	offset := 2
	add := func(value int) int { return value + offset }
	offset = 5
	return add(result)
}

func main() {
	binding := PolicyBinding{"HS256", "original", false}
	println(evaluate(binding, 7), class, Class)
	keyword := Keywords{1, 2, 3, 4, 5, 6}
	println(keyword.constructor, keyword.clone, keyword.class, keyword.match, keyword.namespace, keyword.def)
	δ := Δ{19}
	escaped := _u0394_{23}
	println(δ.Value, escaped.Value)
	anonymous := struct {
		Hash  string
		Count int
	}{"anonymous", 9}
	copied := anonymous
	copied.Count++
	println(anonymous.Hash, anonymous.Count, copied.Count, anonymous == copied)
	var measured Measured = left.New(10)
	println(measured.Value())
	measured = right.New(10)
	println(measured.Value())
	a := left.PolicyBinding{"left", "a", false}
	b := right.PolicyBinding{"right", "b", true}
	println(a.Algorithm, b.Algorithm, a.LegacyString, b.LegacyString)
}

package main

var a = b + 1
var b = f("b", 2)
var c, d = pair()
var total int

func f(name string, v int) int {
	println("init", name)
	return v
}

func pair() (int, int) {
	println("pair")
	return 10, 20
}

func init() {
	println("init1", a, b, c, d)
	total = a + b
}

func init() {
	println("init2", total)
	total *= 2
}

func main() {
	println("main", total)
}

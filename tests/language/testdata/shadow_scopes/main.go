package main

var x = 1

func main() {
	println(x)
	x := 2
	{
		x := 3
		x++
		println(x)
	}
	if x := 10; x > 5 {
		println("if", x)
	} else {
		println("else", x)
	}
	println(x)
	for x := 0; x < 2; x++ {
		x := x * 10
		println("loop", x)
	}
	switch x := "s"; x {
	case "s":
		println("switch", x)
	}
	f := func(x int) int { return x + 1 }
	println(f(x))
	var len = 3
	println(len)
	type int = string
	var s int = "aliased"
	println(s)
}

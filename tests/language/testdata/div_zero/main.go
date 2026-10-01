package main

func div(a, b int32) int32 { return a / b }

func main() {
	println(div(7, 2))
	println(div(-2147483648, -1))
	var z int32
	println(div(1, z))
	println("unreachable")
}

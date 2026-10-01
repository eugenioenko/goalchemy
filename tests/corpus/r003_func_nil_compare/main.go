package main

func main() {
	var f func()
	g := func() {}
	var m map[int]int
	var p *int
	println(f == nil, g != nil, nil == f, m == nil, p == nil)
}

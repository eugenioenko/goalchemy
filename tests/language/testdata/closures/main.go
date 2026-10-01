package main

func counter() (func() int, func()) {
	c := 0
	return func() int { c++; return c }, func() { c = 100 }
}

func compose(f, g func(int) int) func(int) int {
	return func(x int) int { return g(f(x)) }
}

type Op func(int, int) int

func main() {
	inc, reset := counter()
	println(inc(), inc())
	reset()
	println(inc())
	var fns []func() int
	for i := 0; i < 3; i++ {
		fns = append(fns, func() int { return i * 10 })
	}
	for _, v := range []int{7, 8} {
		fns = append(fns, func() int { return v })
	}
	for _, f := range fns {
		print(f(), " ")
	}
	println()
	double := func(x int) int { return x * 2 }
	plus1 := func(x int) int { return x + 1 }
	println(compose(double, plus1)(5), compose(plus1, double)(5))
	var fib func(int) int
	fib = func(n int) int {
		if n < 2 {
			return n
		}
		return fib(n-1) + fib(n-2)
	}
	println(fib(15))
	x := 1
	add := func(d int) { x += d }
	add(5)
	add(10)
	println(x)
	ops := map[string]Op{"add": func(a, b int) int { return a + b }, "mul": func(a, b int) int { return a * b }}
	println(ops["add"](3, 4), ops["mul"](3, 4))
	var nilf func()
	println(nilf == nil)
	acc := 0
	for i := range 4 {
		func() {
			acc += i
		}()
	}
	println(acc)
	s := []int{1, 2, 3}
	modify := func() { s[0] = 99; s = append(s, 4) }
	modify()
	println(s[0], len(s))
	gen := func() func() int {
		n := 0
		return func() int {
			n++
			return n
		}
	}
	g1, g2 := gen(), gen()
	g1()
	println(g1(), g2())
	defer func() { println("recovered nil func call:", recover() != nil) }()
	nilf()
}

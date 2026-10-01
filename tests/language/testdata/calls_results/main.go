package main

func divmod(a, b int) (q, r int) {
	q = a / b
	r = a % b
	return
}

func swap(a, b string) (string, string) { return b, a }

func sum(xs ...int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func pair() (int, int) { return 3, 4 }

func add(a, b int) int { return a + b }

func ack(m, n int) int {
	if m == 0 {
		return n + 1
	}
	if n == 0 {
		return ack(m-1, 1)
	}
	return ack(m-1, ack(m, n-1))
}

func named() (x int) {
	x = 5
	if x > 3 {
		return x * 2
	}
	return
}

func main() {
	q, r := divmod(17, 5)
	println(q, r)
	a, b := swap("x", "y")
	println(a, b)
	println(sum(), sum(1), sum(1, 2, 3), sum([]int{4, 5}...))
	println(add(pair()))
	println(ack(2, 3), named())
	_, r2 := divmod(-17, 5)
	println(r2)
}

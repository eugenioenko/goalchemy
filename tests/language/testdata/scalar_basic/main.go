package main

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func collatz(n uint64) (steps int) {
	for n != 1 {
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
		steps++
	}
	return
}

func main() {
	var a int8 = 127
	a++
	println(a, fib(20), collatz(27))
	x, y := 1, 2
	x, y = y, x
	println(x, y, x < y && y > 0 || false)
	for i := 0; i < 5; i++ {
		if i == 3 {
			continue
		}
		println(i)
	}
}

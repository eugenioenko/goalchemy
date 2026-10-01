package main

func classify(n int) string {
	if n < 0 {
		return "neg"
	} else if n == 0 {
		return "zero"
	} else if n%2 == 0 {
		return "even"
	}
	return "odd"
}

func main() {
	for i := -1; i < 4; i++ {
		println(i, classify(i))
	}
outer:
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if j == 3 {
				continue outer
			}
			if i == 3 {
				break outer
			}
			println(i, j)
		}
	}
	n := 0
	for {
		n++
		if n > 10 {
			break
		}
		if n%3 != 0 {
			continue
		}
		println("multiple", n)
	}
	sum := 0
	for sum < 100 {
		sum += 17
	}
	println(sum)
	for i := range 3 {
		println("range", i)
	}
	var k int8
	for k = 120; k > 0; k += 3 {
		println(k)
	}
	count := 0
	for i := 0; i < 3; i++ {
		for j := i; j < 3; j++ {
			count += i * j
		}
	}
	println(count)
}

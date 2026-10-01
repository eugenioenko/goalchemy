package main

func f() (n int, s string) {
	defer func() {
		n++
		s += "!"
	}()
	n, s = 1, "a"
	return n * 10, s + "b"
}

func g() (err string) {
	defer func() {
		if recover() != nil {
			err = "recovered"
		}
	}()
	var m map[int]int
	m[1] = 1
	return "unreached"
}

func main() {
	println(f())
	println(g())
}

package main

func name(n int) string {
	switch n {
	case 0:
		return "zero"
	case 1, 2, 3:
		return "small"
	case 4:
		fallthrough
	case 5:
		return "four-or-five"
	default:
		return "big"
	}
}

func trace(s string, v int) int {
	println("eval", s)
	return v
}

func main() {
	for i := 0; i < 7; i++ {
		println(i, name(i))
	}
	x := 3
	switch {
	case x > 5:
		println("gt5")
	case x > 2:
		println("gt2")
		fallthrough
	case x > 100:
		println("fell")
	case x > 1:
		println("gt1")
	}
	switch y := x * 2; y {
	case trace("a", 1), trace("b", 6), trace("c", 7):
		println("matched", y)
	}
	switch {
	default:
		println("default first")
	case x == 3:
		println("x is 3")
	}
	for i := 0; i < 3; i++ {
		switch i {
		case 1:
			continue
		case 2:
			break
		}
		println("after switch", i)
	}
loop:
	for i := 0; ; i++ {
		switch {
		case i == 2:
			break loop
		}
		println("loop", i)
	}
	var u uint8 = 200
	switch u + 100 {
	case 44:
		println("wrapped")
	}
}

package main

func order() {
	for i := 0; i < 3; i++ {
		defer println("defer", i)
	}
	println("body")
}

func args() {
	x := 1
	defer println("deferred x =", x)
	x = 2
	println("x =", x)
}

func named() (r int) {
	defer func() { r *= 2 }()
	return 21
}

func safeDiv(a, b int) (q int, err string) {
	defer func() {
		if e := recover(); e != nil {
			err = "recovered: " + e.(error).Error()
		}
	}()
	return a / b, ""
}

func repanic() {
	defer func() {
		r := recover()
		println("outer got", r.(string))
	}()
	defer func() {
		panic("second")
	}()
	panic("first")
}

func helper() any { return recover() }

func indirect() (got bool) {
	defer func() {
		got = helper() != nil
		recover()
	}()
	panic("x")
}

func nilPanic() {
	defer func() {
		r := recover()
		println("nil panic recovered:", r != nil)
		if e, ok := r.(error); ok {
			println(e.Error())
		}
	}()
	panic(nil)
}

type T struct{ n int }

func (t *T) close() { println("close", t.n) }

func methods() {
	t := &T{1}
	defer t.close()
	t = &T{2}
	defer t.close()
}

func deferInLoop() (s string) {
	for _, c := range []string{"a", "b", "c"} {
		defer func() { s += c }()
	}
	return "x"
}

func main() {
	order()
	args()
	println(named())
	println(safeDiv(7, 2))
	println(safeDiv(1, 0))
	repanic()
	println("indirect helper recovered:", indirect())
	nilPanic()
	methods()
	println(deferInLoop())
	println(recover() == nil)
	defer println("main deferred runs before crash")
	var m map[string]int
	m["boom"] = 1
}

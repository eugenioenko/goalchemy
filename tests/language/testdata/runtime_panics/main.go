package main

type T struct{ x int }

type I interface{ M() }

type E struct{}

func (E) M() {}

type F struct{ x int }

func (f F) M() { println(f.x) }

func try(name string, f func()) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			println(name, "->", e.Error())
		} else {
			println(name, "-> non-error")
		}
	}()
	f()
}

func main() {
	s := []int{1, 2, 3}
	var u uint64 = 18446744073709551615
	n := -1
	five := 5
	try("index", func() { _ = s[five] })
	try("index neg", func() { _ = s[n] })
	try("index huge", func() { _ = s[u] })
	try("slice hi", func() { _ = s[:five] })
	try("slice lo>hi", func() { lo := 3; hi := 2; _ = s[lo:hi] })
	try("slice3 max", func() { _ = s[0:1:five] })
	try("slice3 hi>max", func() { hi := 3; _ = s[0:hi:2] })
	try("string slice", func() { str := "abc"; _ = str[1:five] })
	try("array index", func() { var a [3]int; _ = a[five] })
	try("nil map", func() { var m map[string]int; m["a"] = 1 })
	try("div", func() { z := 0; _ = 10 / z })
	try("rem", func() { var z int8; _ = int8(1) % z })
	try("shift", func() { _ = 1 << n })
	try("nil ptr", func() { var p *T; _ = p.x })
	try("nil iface", func() { var i I; i.M() })
	try("assert", func() { var i any = 1; _ = i.(string) })
	try("assert iface", func() { var i any = T{}; _ = i.(I) })
	try("assert nil", func() { var i any; _ = i.(int) })
	try("assert named", func() { var i I = E{}; _ = i.(interface{ N() }) })
	try("uncomparable", func() { var a, b any = []int{}, []int{}; _ = a == b })
	try("hash", func() { m := map[any]int{}; m[[]int{}] = 1 })
	try("make", func() { _ = make([]int, n) })
	try("make cap", func() { _ = make([]int, 3, 1+n+1) })
	try("conv", func() { _ = [4]int(s) })
	try("value method expr nil", func() { var p *F; f := (*F).M; f(p) })
	try("custom", func() { panic(T{}) })
	try("closure nil", func() { var f func(); f() })
}

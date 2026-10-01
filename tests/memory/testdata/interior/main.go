package main

type T struct {
	a, b int
	s    string
}

type Shape interface{ Area() int }

type Rect struct{ w, h int }

func (r Rect) Area() int { return r.w * r.h }

func churn() int {
	n := 0
	for i := 0; i < 200000; i++ {
		x := make([]int, 3)
		n += len(x)
	}
	return n
}

func fieldPtr() *int {
	t := &T{a: 1, b: 2, s: "x"}
	return &t.b
}

func tailOf() []int {
	big := make([]int, 10)
	for i := range big {
		big[i] = i
	}
	return big[8:]
}

func counter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

func main() {
	p := fieldPtr()
	tail := tailOf()
	var sh Shape = Rect{w: 3, h: 4}
	next := counter()
	next()
	m := map[string][]int{"a": {1, 2, 3}}
	arr := [4]int{5, 6, 7, 8}
	sub := arr[1:3]
	defer func() {
		churn()
		println("deferred", *p, tail[1], sh.Area(), m["a"][2])
	}()
	n := churn()
	println(n > 0, *p, tail[0], tail[1], sh.Area(), next(), m["a"][1], sub[0], sub[1])
}

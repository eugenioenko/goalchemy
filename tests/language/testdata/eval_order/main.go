package main

var trace []string

func t(name string, v int) int {
	trace = append(trace, name)
	return v
}

func dump() {
	for _, s := range trace {
		print(s, " ")
	}
	println()
	trace = trace[:0]
}

func pair() (int, int) {
	trace = append(trace, "pair")
	return 1, 2
}

type S struct{ a, b int }

func main() {
	_ = t("a", 1) + t("b", 2)*t("c", 3)
	dump()
	arr := []int{0, 0, 0}
	arr[t("idx", 1)] = t("val", 5)
	dump()
	println(arr[1])
	i := 0
	i, arr[i] = 2, 9
	println(i, arr[0], arr[2])
	x, y := 1, 2
	x, y = y, x
	println(x, y)
	m := map[int]int{}
	m[t("k", 1)] += t("v", 10)
	dump()
	println(m[1])
	f := func(a, b, c int) int { return a + b + c }
	_ = f(t("1", 1), t("2", 2), t("3", 3))
	dump()
	s := S{a: t("fa", 1), b: t("fb", 2)}
	dump()
	println(s.a, s.b)
	a, b := pair()
	println(a, b)
	j := 0
	sl := []int{10, 20, 30}
	j, sl[j] = 1, 100
	println(j, sl[0], sl[1])
	if t("cond1", 0) == 1 || t("cond2", 1) == 1 {
		dump()
	}
	vals := []int{t("e0", 0), t("e1", 1)}
	dump()
	println(len(vals))
	p := &S{}
	p.a, p = 5, &S{7, 8}
	println(p.a)
	k := 1
	k += func() int { k = 100; return 1 }()
	println(k)
}

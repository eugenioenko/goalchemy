package main

func show(name string, s []int) {
	print(name, " len=", len(s), " [")
	for i, v := range s {
		if i > 0 {
			print(" ")
		}
		print(v)
	}
	println("]")
}

func main() {
	s := make([]int, 3, 10)
	t := s[1:2]
	t[0] = 5
	show("s", s)
	t = append(t, 6)
	show("s", s)
	show("t", t)
	u := s[:2:2]
	u = append(u, 99)
	u[0] = -1
	show("s", s)
	show("u", u)
	var n []int
	e := []int{}
	println(n == nil, e == nil, len(n), cap(n))
	n = append(n, 1)
	println(n == nil, len(n))
	x := []int{1, 2, 3, 4, 5}
	copy(x[1:], x)
	show("x", x)
	y := []int{1, 2, 3, 4, 5}
	copy(y, y[2:])
	show("y", y)
	z := []int{1, 2, 3, 4, 5}
	z = append(z[:1], z[2:]...)
	show("z", z)
	b := []byte("hello")
	b[0] = 'j'
	println(string(b), len(b))
	b = append(b, " world"...)
	println(string(b))
	nested := [][]int{{1}, {2, 3}}
	nested[1] = append(nested[1], 4)
	show("n1", nested[1])
	empty := x[5:]
	println(len(empty), empty == nil)
	r := []rune("héllo")
	println(len(r), r[1], string(r[1:3]))
	full := x[1:3:4]
	println(len(full), cap(full))
	clear(full)
	show("x", x)
	var ss []string
	ss = append(ss, "a", "b")
	ss2 := append([]string(nil), ss...)
	ss2[0] = "z"
	println(ss[0], ss2[0], len(ss2))
}

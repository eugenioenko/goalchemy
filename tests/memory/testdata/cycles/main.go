package main

type Node struct {
	next *Node
	val  int
	data []int
}

type Ring struct {
	a, b *Node
	f    func() int
}

func main() {
	total := 0
	for i := 0; i < 300000; i++ {
		a := &Node{val: i}
		b := &Node{val: i + 1, next: a}
		a.next = b
		a.data = make([]int, 4)
		r := &Ring{a: a, b: b}
		r.f = func() int { return r.a.val + r.b.next.val }
		m := map[int]*Node{0: a, 1: b}
		m[2] = m[0].next.next
		a.data[0] = len(m)
		total += (r.f() + a.data[0]) % 7
	}
	println("total", total)
}

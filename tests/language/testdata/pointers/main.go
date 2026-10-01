package main

type Node struct {
	Val  int
	Next *Node
}

type Counter struct{ n int }

func (c *Counter) Inc() int { c.n++; return c.n }
func (c Counter) Get() int  { return c.n }

func incr(p *int) { *p++ }

func newInt(v int) *int { return &v }

func main() {
	x := 1
	p := &x
	incr(p)
	incr(&x)
	println(x, *p, p == &x)
	q := newInt(5)
	r := newInt(5)
	println(*q == *r, q == r)
	var head *Node
	for i := 3; i > 0; i-- {
		head = &Node{i, head}
	}
	for n := head; n != nil; n = n.Next {
		print(n.Val, " ")
	}
	println()
	c := Counter{}
	c.Inc()
	cp := &c
	cp.Inc()
	println(c.Get(), cp.Get(), c.n)
	type S struct{ A, B int }
	s := S{1, 2}
	pa := &s.A
	s = S{10, 20}
	*pa += 1
	println(s.A, s.B, *pa)
	pp := &p
	**pp = 50
	println(x)
	n := new(int)
	*n = 7
	println(*n)
	ns := new(S)
	ns.B = 3
	println(ns.A, ns.B)
	var nilp *Node
	defer func() {
		r := recover()
		println("recovered", r != nil)
		if e, ok := r.(error); ok {
			println(e.Error())
		}
	}()
	println(nilp.Val)
}

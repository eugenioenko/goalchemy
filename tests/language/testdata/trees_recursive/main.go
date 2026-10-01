package main

type Tree struct {
	Left, Right *Tree
	Val         int
}

func (t *Tree) Insert(v int) *Tree {
	if t == nil {
		return &Tree{Val: v}
	}
	if v < t.Val {
		t.Left = t.Left.Insert(v)
	} else {
		t.Right = t.Right.Insert(v)
	}
	return t
}

func (t *Tree) Walk(f func(int)) {
	if t == nil {
		return
	}
	t.Left.Walk(f)
	f(t.Val)
	t.Right.Walk(f)
}

func (t *Tree) Height() int {
	if t == nil {
		return 0
	}
	return 1 + max(t.Left.Height(), t.Right.Height())
}

type Expr interface{ Eval() int }

type Lit int
type Add struct{ L, R Expr }
type Mul struct{ L, R Expr }

func (l Lit) Eval() int { return int(l) }
func (a Add) Eval() int { return a.L.Eval() + a.R.Eval() }
func (m Mul) Eval() int { return m.L.Eval() * m.R.Eval() }

type Graph map[string][]string

func (g Graph) reach(from string, seen map[string]bool) {
	if seen[from] {
		return
	}
	seen[from] = true
	for _, n := range g[from] {
		g.reach(n, seen)
	}
}

func main() {
	var root *Tree
	for _, v := range []int{5, 3, 8, 1, 4, 9, 7} {
		root = root.Insert(v)
	}
	root.Walk(func(v int) { print(v, " ") })
	println(root.Height())
	e := Add{Lit(2), Mul{Lit(3), Add{Lit(1), Lit(4)}}}
	println(e.Eval())
	g := Graph{"a": {"b", "c"}, "b": {"d"}, "c": {"a"}, "e": {"a"}}
	seen := map[string]bool{}
	g.reach("a", seen)
	println(len(seen), seen["d"], seen["e"])
	type cyc struct {
		next *cyc
		n    int
	}
	x := &cyc{n: 1}
	y := &cyc{n: 2, next: x}
	x.next = y
	println(x.next.next.n, y.next.next.next.n)
}

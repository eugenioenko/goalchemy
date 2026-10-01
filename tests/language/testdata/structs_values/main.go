package main

type Point struct{ X, Y int }

type Rect struct {
	Min, Max Point
	Tags     [2]string
	Data     []int
}

func (p Point) Add(q Point) Point { return Point{p.X + q.X, p.Y + q.Y} }

func move(r Rect) Rect {
	r.Min.X += 100
	r.Tags[0] = "moved"
	r.Data[0] = 42
	return r
}

type Pair struct {
	a int8
	b string
}

func main() {
	a := Rect{Min: Point{1, 2}, Max: Point{3, 4}, Tags: [2]string{"x", "y"}, Data: []int{1, 2}}
	b := a
	b.Min.X = 9
	b.Tags[1] = "z"
	println(a.Min.X, b.Min.X, a.Tags[1], b.Tags[1])
	c := move(a)
	println(a.Min.X, c.Min.X, a.Tags[0], c.Tags[0], a.Data[0], c.Data[0])
	p := Point{1, 1}.Add(Point{2, 3})
	println(p.X, p.Y, p == Point{3, 4}, p != Point{3, 4})
	var z Rect
	println(z.Min.X, z.Tags[0] == "", z.Data == nil, len(z.Data))
	anon := struct {
		Name string
		N    int
	}{"anon", 7}
	anon2 := anon
	anon2.N++
	println(anon.Name, anon.N, anon2.N, anon == anon2)
	pairs := []Pair{{1, "a"}, {2, "b"}}
	q := pairs[0]
	q.a = 99
	pairs[1].b = "B"
	println(pairs[0].a, q.a, pairs[1].b)
	arr := [3]Point{{1, 1}, {2, 2}}
	arr2 := arr
	arr2[0].X = 50
	println(arr[0].X, arr2[0].X, arr == arr2, arr[2].Y)
	var pp *Point = &Point{5, 6}
	cp := *pp
	cp.X = 0
	println(pp.X, cp.X)
	type local struct{ v int }
	l1, l2 := local{1}, local{1}
	println(l1 == l2)
}

package main

type Shape interface {
	Area() int
	Perimeter() int
}

type Named interface{ Name() string }

type NamedShape interface {
	Shape
	Named
}

type Square struct{ s int }

func (q Square) Area() int      { return q.s * q.s }
func (q Square) Perimeter() int { return 4 * q.s }
func (q Square) Name() string   { return "square" }

type Circle struct{ r int }

func (c *Circle) Area() int      { return 3 * c.r * c.r }
func (c *Circle) Perimeter() int { return 6 * c.r }

type MyErr struct{ code int }

func (e *MyErr) Error() string { return "myerr" }

func find(fail bool) error {
	var e *MyErr
	if fail {
		e = &MyErr{1}
	}
	return e
}

func describe(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case int, int8:
		return "integer"
	case string:
		return "string:" + x
	case Shape:
		return "shape"
	case error:
		return "error:" + x.Error()
	case []int:
		return "slice"
	default:
		return "other"
	}
}

func main() {
	shapes := []Shape{Square{2}, &Circle{1}}
	total := 0
	for _, s := range shapes {
		total += s.Area() + s.Perimeter()
	}
	println(total)
	var ns NamedShape = Square{3}
	var sh Shape = ns
	println(ns.Name(), sh.Area())
	_, isNamed := sh.(Named)
	_, circNamed := shapes[1].(Named)
	println(isNamed, circNamed)
	sq, ok := sh.(Square)
	println(sq.s, ok)
	_, ok = sh.(*Circle)
	println(ok)
	err := find(false)
	println(err == nil, err != nil)
	println(describe(nil), describe(3), describe(int8(3)), describe("hi"), describe(Square{1}), describe(&MyErr{}), describe([]int{}), describe(1 == 1))
	var a, b any = 1, 1
	var c any = int64(1)
	println(a == b, a == c, a == 1, c == int64(1))
	var x, y any = Square{1}, Square{1}
	println(x == y, x != Square{2})
	m := map[any]int{}
	m[Square{1}] = 1
	m[Square{1}]++
	println(m[Square{1}], len(m))
	var empty any
	println(empty == nil)
	defer func() {
		r := recover()
		println("recovered:", r.(error).Error())
		defer func() {
			println("second:", recover().(error).Error())
		}()
		var u1, u2 any = []int{1}, []int{1}
		println(u1 == u2)
	}()
	var s Shape = &Circle{2}
	_ = s.(Square)
}

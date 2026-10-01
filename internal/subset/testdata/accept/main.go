package main

import "goalchemy/lib/errors"

type Shape interface {
	Area() int
	Name() string
}

type Base struct{ id int }

func (b Base) Name() string { return "base" }

type Rect struct {
	Base
	W, H int
	tags []string
	meta map[string]int
}

func (r *Rect) Area() int { return r.W * r.H }

type Celsius int8

var counter = initCounter()

func initCounter() int { return 3 }

func init() { counter++ }

func divide(a, b int) (q int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("recovered")
		}
	}()
	return a / b, nil
}

func sum(xs ...int) (t int) {
	for _, x := range xs {
		t += x
	}
	return
}

func main() {
	r := &Rect{W: 2, H: 3}
	var s Shape = r
	if rr, ok := s.(*Rect); ok {
		rr.W++
	}
	switch v := s.(type) {
	case *Rect:
		println(v.Area())
	default:
	}
	arr := [3]int{1, 2, 3}
	sl := arr[:]
	sl = append(sl, 4)
	m := map[string][]int{"a": {1}}
	for k, v := range m {
		println(k, len(v))
	}
	for i := range 3 {
		defer println(i)
	}
	var fns []func() int
	for i := 0; i < 3; i++ {
		fns = append(fns, func() int { return i })
	}
outer:
	for _, c := range "héllo" {
		switch {
		case c == 'l':
			continue outer
		case c > 1000:
			break outer
		default:
			fallthrough
		case c == 0:
		}
	}
	p := &r.W
	*p = 7
	q, err := divide(1, 0)
	println(q, err != nil, sum(1, 2, 3), Celsius(5), counter, min(1, 2), string(rune(65)))
	b := []byte("abc")
	println(string(b[1:]), len(b), cap(b))
	clear(m)
}

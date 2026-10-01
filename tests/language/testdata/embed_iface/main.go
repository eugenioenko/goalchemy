package main

type Writer interface{ Write(s string) int }

type Console struct{ prefix string }

func (c *Console) Write(s string) int {
	println(c.prefix + s)
	return len(s)
}

type Buffered struct {
	Writer
	n int
}

func (b *Buffered) Flush() { println("flush", b.n) }

type Base struct{ id int }

func (b *Base) ID() int     { return b.id }
func (b Base) Kind() string { return "base" }

type Derived struct {
	*Base
	name string
}

type IDer interface {
	ID() int
	Kind() string
}

type Stack struct{ items []int }

func (s *Stack) Push(v int) { s.items = append(s.items, v) }
func (s *Stack) Pop() int {
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v
}

func main() {
	b := &Buffered{Writer: &Console{"> "}}
	n := b.Write("hello")
	b.n += n
	b.Flush()
	var w Writer = b
	w.Write("via iface")
	write := w.Write
	write("method value")
	defer w.Write("deferred iface call")
	d := Derived{&Base{7}, "d"}
	var i IDer = d
	println(i.ID(), i.Kind())
	var ip IDer = &d
	println(ip.ID(), ip.Kind())
	d.Base = &Base{8}
	println(i.ID(), d.ID())
	s := &Stack{}
	push := s.Push
	for k := 0; k < 3; k++ {
		push(k)
	}
	pop := (*Stack).Pop
	println(pop(s), pop(s), len(s.items))
	var nilW Writer
	defer func() {
		println("recovered:", recover() != nil)
	}()
	f := nilW.Write
	f("never")
}

package main

type Animal struct{ Name string }

func (a Animal) Speak() string    { return a.Name + " makes a sound" }
func (a *Animal) Rename(n string) { a.Name = n }

type Dog struct {
	Animal
	Breed string
}

func (d Dog) Speak() string { return d.Name + " barks" }

type Puppy struct {
	*Dog
	Age int
}

type Logger struct{ prefix string }

func (l *Logger) Log(s string) { println(l.prefix + s) }

type Service struct {
	Logger
	name string
}

type Num int

func (n Num) Double() Num   { return n * 2 }
func (n *Num) Incr()        { *n++ }
func (n Num) Add(m Num) Num { return n + m }

func apply(f func(Num) Num, v Num) Num { return f(v) }

func main() {
	d := Dog{Animal{"Rex"}, "lab"}
	println(d.Speak(), "|", d.Animal.Speak())
	d.Rename("Max")
	println(d.Name, d.Animal.Name)
	p := Puppy{&d, 1}
	p.Rename("Tiny")
	println(d.Name, p.Speak(), p.Breed)
	s := Service{Logger{"[svc] "}, "x"}
	s.Log("hello")
	sp := &s
	sp.Log("via pointer")
	var n Num = 5
	n.Incr()
	println(n, n.Double())
	f := n.Double
	n = 100
	println(f(), n.Double())
	g := Num.Add
	h := (*Num).Incr
	h(&n)
	println(g(1, 2), n)
	println(apply(Num.Double, 21))
	speak := d.Speak
	d.Name = "Changed"
	println(speak())
	rn := d.Rename
	rn("Final")
	println(d.Name)
	var ap *Animal = &d.Animal
	ap.Rename("ViaPtr")
	println(d.Name, ap.Speak())
}

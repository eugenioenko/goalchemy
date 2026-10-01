package main

type Set map[string]bool

func (s Set) Add(k string) Set  { s[k] = true; return s }
func (s Set) Has(k string) bool { return s[k] }

func main() {
	s := Set{}
	s.Add("a").Add("b")
	var n Set
	println(s.Has("a"), s.Has("z"), len(s), n == nil, n.Has("a"))
}

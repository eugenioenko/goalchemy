package main

// goalchemy:unordered

type key struct {
	a int
	b string
}

func main() {
	m := map[string]int{"one": 1, "two": 2}
	m["three"] = 3
	v, ok := m["two"]
	println(v, ok, m["missing"], len(m))
	delete(m, "one")
	delete(m, "nope")
	_, ok = m["one"]
	println(ok, len(m))
	m["two"] += 10
	m["four"]++
	println(m["two"], m["four"])
	var nm map[string]bool
	println(nm["x"], len(nm))
	delete(nm, "x")
	sk := map[key][]int{}
	sk[key{1, "a"}] = append(sk[key{1, "a"}], 5)
	sk[key{1, "a"}] = append(sk[key{1, "a"}], 6)
	println(len(sk[key{1, "a"}]), len(sk[key{2, "a"}]))
	ik := map[any]string{1: "int", "1": "string", int8(1): "int8"}
	println(ik[1], ik["1"], ik[int8(1)], ik[int16(1)] == "")
	alias := m
	alias["five"] = 5
	println(m["five"], len(alias) == len(m))
	type point struct{ x, y int }
	pm := map[string]point{"p": {1, 2}}
	p := pm["p"]
	p.x = 100
	println(pm["p"].x, p.x)
	total := 0
	for k, v := range m {
		println("entry", k, v)
		total += v
	}
	println("total", total)
	clear(m)
	println(len(m))
	counts := map[rune]int{}
	for _, r := range "banana" {
		counts[r]++
	}
	println(counts['a'], counts['b'], counts['n'])
}

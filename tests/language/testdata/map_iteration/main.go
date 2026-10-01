package main

// goalchemy:golden

func main() {
	m := map[string]int{}
	for _, k := range []string{"c", "a", "b", "e", "d"} {
		m[k] = len(k)
	}
	for k := range m {
		print(k, " ")
	}
	println()
	m["a"] = 10
	delete(m, "b")
	m["b"] = 20
	for k, v := range m {
		print(k, "=", v, " ")
	}
	println()
	for k := range m {
		if k == "c" {
			delete(m, "e")
			m["z"] = 1
			m["d"] = 99
		}
		print(k, ":", m[k], " ")
	}
	println()
	println(len(m))
}

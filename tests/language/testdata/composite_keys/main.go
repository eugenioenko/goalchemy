package main

// goalchemy:unordered

type Pt struct{ X, Y int }

type Key struct {
	P    Pt
	Tags [2]string
}

func main() {
	grid := map[Pt]string{{0, 0}: "origin", {1, 2}: "a"}
	grid[Pt{1, 2}] += "!"
	println(grid[Pt{0, 0}], grid[Pt{1, 2}], len(grid))
	k1 := Key{Pt{1, 1}, [2]string{"x", "y"}}
	k2 := k1
	m := map[Key]int{k1: 1}
	m[k2]++
	k2.Tags[1] = "z"
	m[k2] = 5
	println(m[k1], m[k2], len(m))
	arr := map[[3]int]bool{{1, 2, 3}: true}
	println(arr[[3]int{1, 2, 3}], arr[[3]int{3, 2, 1}])
	ik := map[any]int{}
	ik[Pt{1, 1}] = 1
	ik[[2]int{1, 1}] = 2
	ik[Pt{1, 1}] += 10
	var p1, p2 any = Pt{1, 1}, Pt{1, 1}
	println(ik[Pt{1, 1}], ik[[2]int{1, 1}], len(ik), p1 == p2)
	ptrs := map[*Pt]int{}
	a, b := &Pt{}, &Pt{}
	ptrs[a] = 1
	ptrs[b] = 2
	ptrs[a]++
	println(ptrs[a], ptrs[b], len(ptrs))
	type pair struct {
		a any
		b string
	}
	pm := map[pair]int{{1, "x"}: 1}
	pm[pair{1, "x"}]++
	pm[pair{int8(1), "x"}]++
	println(pm[pair{1, "x"}], len(pm))
	for k, v := range grid {
		println("grid", k.X, k.Y, v)
	}
}

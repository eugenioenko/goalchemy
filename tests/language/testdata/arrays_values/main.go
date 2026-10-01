package main

func sum(a [4]int) int {
	t := 0
	for _, v := range a {
		t += v
	}
	a[0] = 1000
	return t
}

func fill(p *[4]int) {
	for i := range p {
		p[i] = i * i
	}
}

func main() {
	var a [4]int
	fill(&a)
	println(a[0], a[1], a[2], a[3], sum(a), a[0])
	b := [...]string{2: "two", 0: "zero"}
	println(len(b), b[0], b[1] == "", b[2])
	grid := [2][3]int{{1, 2, 3}, {4, 5, 6}}
	g2 := grid
	g2[1][2] = 60
	println(grid[1][2], g2[1][2], grid == g2)
	for i, row := range grid {
		row[0] = -1
		println(i, row[0], grid[i][0])
	}
	arr := [3]int{1, 2, 3}
	for i, v := range arr {
		arr[2] = 100
		println(i, v)
	}
	s := arr[:]
	s[0] = 7
	println(arr[0], len(s), cap(s))
	p := &arr
	println(len(p), p[1])
	var idx = 5
	defer func() {
		println("recovered:", recover() != nil)
	}()
	println(arr[idx%3])
	println(arr[idx])
}

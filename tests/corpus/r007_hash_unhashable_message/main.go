package main

type K struct{ v any }

func main() {
	defer func() { println(recover().(error).Error()) }()
	m := map[K]int{}
	m[K{[]int{1}}] = 1
}

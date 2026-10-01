package main

func main() {
	x := 1
	for _, s := range []int{1, 63, -1} {
		println(x << s)
	}
}

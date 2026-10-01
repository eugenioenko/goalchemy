package main

// goalchemy:golden

func main() {
	var s []int
	for i := 0; i < 10; i++ {
		s = append(s, i)
		print(cap(s), " ")
	}
	println()
	t := make([]int, 0, 3)
	t = append(t, 1, 2, 3, 4)
	println(len(t), cap(t))
	u := append([]int{1}, 2, 3, 4, 5, 6)
	println(len(u), cap(u))
	b := []byte("abc")
	println(cap(b), cap([]rune("héllo")))
	b = append(b, "defgh"...)
	println(len(b), cap(b))
}

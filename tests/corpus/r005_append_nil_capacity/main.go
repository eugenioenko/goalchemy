package main

// goalchemy:golden

func main() {
	var s []string
	s = append(s, "a")
	println(len(s), cap(s))
	r := []rune("héllo")
	println(len(r), cap(r))
}

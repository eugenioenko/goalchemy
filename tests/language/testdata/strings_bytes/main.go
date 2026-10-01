package main

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func main() {
	s := "héllo, 世界"
	println(len(s), s[0], s[1], s[1:3] == "é", s[7:])
	for i, r := range s {
		print(i, ":", r, " ")
	}
	println()
	bad := "a\xffb\xe2\x82"
	for i, r := range bad {
		print(i, ":", r, " ")
	}
	println(len(bad))
	println(reverse("abc"), reverse(s))
	b := []byte(s)
	b[0] = 'H'
	println(string(b[:2]), s[:1])
	var acc string
	for i := 0; i < 5; i++ {
		acc += string(rune('a' + i))
	}
	println(acc, acc < "abcdf", "Z" < "a", "" < "a", "ab" > "a")
	println(string(rune(0x10FFFF+1)) == "�", string(rune(-1)), string(rune(0xD800)) == "�")
	var n int64 = 4294967361
	println(string(rune(n)), len(string(rune(n))))
	runes := []rune{72, 105, -5}
	println(string(runes))
	println(string([]byte{0xe4, 0xb8, 0x96}), len([]rune("\xe4\xb8")))
	x := "abc"
	y := x[1:]
	println(y, len(y), x[:0] == "", x[3:] == "")
	idx := 3
	defer func() { println("recovered:", recover().(error).Error()) }()
	println(x[idx])
}

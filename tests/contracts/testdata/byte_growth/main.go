package main

type Octet uint8

func main() {
	b := make([]byte, 2, 2)
	b[0], b[1] = 255, 128
	b = append(b, 3)
	v := b[:4]
	println(len(v), v[0], v[1], v[2], v[3])
	n := []Octet{1, 2}
	n = append(n, 3)
	println(n[:4][3])
	n = append(n[:3:3], n[:2]...)
	println(n[:6][5])
}

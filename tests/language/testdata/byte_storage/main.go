package main

type Octet uint8
type Bytes []byte
type Octets []Octet
type Alias = uint8
type Holder struct {
	B Bytes
	A [4]Octet
}

func show(b []byte) {
	print(len(b), ":")
	for _, x := range b {
		print(" ", x)
	}
	println()
}
func caught(f func()) {
	defer func() { println("panic", recover() != nil) }()
	f()
}
func escaped() (*[4]byte, []byte) {
	a := [4]byte{1, 2, 3, 4}
	return &a, a[1:]
}

func main() {
	var n Bytes
	var u []uint8
	e := Bytes{}
	clear(n)
	clear(e)
	println(copy(n, "abc"), copy(n, e), copy(e, n))
	println(n == nil, e == nil, len(n), cap(n), append(n) == nil, append(n, u...) == nil)
	println(n[:0:0] == nil, make([]byte, 0) == nil)
	n = append(n[:0], 255, 0, 128)
	show(n)
	z := make(Bytes, 0, 8)
	show(z[:8])
	z = append(z, 1, 2, 3, 4, 5)
	clear(z[1:1])
	clear(z[8:8])
	view := z[1:3:6]
	view[0] = 200
	view = append(view, 9, 10)
	show(z)
	show(view)
	show(z[:8])
	full := z[:2:2]
	grown := append(full, 7)
	grown[0] = 99
	show(z)
	show(grown)
	for _, b := range [][]byte{{1, 2, 3, 4, 5}, {1, 2, 3, 4, 5}} {
		copy(b[1:], b)
		show(b)
		copy(b, b[2:])
		show(b)
	}
	x := []byte{1, 2, 3, 4, 5, 6}
	y := append(x[:2], x[1:5]...)
	show(x)
	show(y)
	y = append(x[1:3:3], x[:4]...)
	show(x)
	show(y)
	clear(x[1:4])
	show(x)
	a := [4]byte{0, 128, 255, 65}
	as := a[:]
	acopy := a
	as[0] = 7
	println(a[0], acopy[0], a == acopy)
	arr := [4]byte(as)
	as[1] = 12
	println(arr[1], a[1])
	a = [4]byte{9, 8, 7, 6}
	show(as)
	ptr, escapedView := escaped()
	ptr[1] = 250
	println(escapedView[0])
	escapedView[1] = 240
	println(ptr[2])
	var dyn any = as
	dynView := dyn.([]byte)
	dynView[2] = 33
	println(a[2])
	var h Holder
	h.B = Bytes{1, 2}
	h.A = [4]Octet{255, 2, 3, 4}
	hs := h.A[:]
	hc := h
	h.A = [4]Octet{5, 6, 7, 8}
	println(hs[0], hc.A[0])
	p := &h.B
	*p = append(*p, 3)
	show(h.B)
	var boxed any = h
	bh := boxed.(Holder)
	bh.B[0] = 42
	bh.A[0] = 99
	println(h.B[0], h.A[0], bh.A[0])
	capture := func() { h.B[1]++; h.B = append(h.B, 4) }
	capture()
	show(h.B)
	var named Octets
	named = append(named, 255, 0, 128)
	named[0]++
	println(named[0], named[1], named[2])
	aliases := []Alias{255, 128}
	show(aliases)
	bin := "\x00\xff\x80\xc0\xafA"
	b := []byte(bin)
	saved := string(b)
	b[0] = 44
	println(saved == bin, string(b) == bin, len(saved))
	b2 := []byte(saved)
	b2[1] = 1
	println(saved == bin, b[1], b2[1])
	b = append(b[:1:1], bin...)
	show(b)
	copy(b[2:], bin)
	show(b)
	println(string([]byte(nil)) == "", []byte("") == nil)
	zero := [0]byte(n[:0])
	println(len(zero), cap(zero), zero[:] == nil)
	caught(func() { println(n[len(n)]) })
	caught(func() { n[len(n)] = 1 })
	caught(func() { show(n[:cap(n)+1]) })
	caught(func() { k := 0; show(n[:1:k]) })
	caught(func() { v := [4]byte(n); println(v[0]) })
	caught(func() { k := -1; show(make([]byte, k)) })
	ints := []int{1000, -1, 2}
	ints = append(ints, ints...)
	println(ints[0], ints[1], ints[4])
	runes := []rune("é")
	println(runes[0], string(runes))
}

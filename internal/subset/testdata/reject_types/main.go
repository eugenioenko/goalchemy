package main

type T struct {
	f float64
}

var c complex128 // want GCS004

func g(p uintptr) {} // want GCS004

func main() {
	ch := make(chan int) // want GCS011
	_ = ch
	x := 1.5
	_ = x
	const ok = 2.5 * 2
	var n int = ok
	println(n)
	println(real(c)) // want GCS007
}

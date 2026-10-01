package main

func work() {}

func seq(yield func(int) bool) {}

func main() {
	go work() // want GCS011
	i := 0
loop:
	i++
	if i < 3 {
		goto loop // want GCS005
	}
	for x := range seq { // want GCS005
		_ = x
	}
	var p *int
	println(p) // want GCS006
}

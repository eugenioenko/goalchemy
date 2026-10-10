package main

// goalchemy:gate cooperative

import "github.com/eugenioenko/goalchemy/std/time"

type napper struct{ d time.Duration }

func (n napper) nap(out chan string, name string) {
	time.Sleep(n.d)
	out <- name
}

func main() {
	out := make(chan string)
	for _, w := range []struct {
		name string
		d    time.Duration
	}{{"slow", 60 * time.Millisecond}, {"fast", 20 * time.Millisecond}, {"medium", 40 * time.Millisecond}} {
		go func() {
			time.Sleep(w.d)
			out <- w.name
		}()
	}
	for i := 0; i < 3; i++ {
		println(<-out)
	}
	go napper{30 * time.Millisecond}.nap(out, "method")
	go napper{0}.nap(out, "zero")
	println(<-out, <-out)
	d, err := time.ParseDuration("10ms")
	if err != nil {
		panic(err)
	}
	time.Sleep(d)
	time.Sleep(-time.Second)
	println("done", d.String())
}

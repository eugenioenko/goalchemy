package main

// goalchemy:gate cooperative

func try(name string, f func()) {
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			println(name, "->", e.Error())
		}
	}()
	f()
}

func main() {
	try("send closed", func() {
		ch := make(chan int, 1)
		close(ch)
		ch <- 1
	})
	try("close closed", func() {
		ch := make(chan int)
		close(ch)
		close(ch)
	})
	try("close nil", func() {
		var ch chan int
		close(ch)
	})
	try("make negative", func() {
		n := -1
		_ = make(chan int, n)
	})
	waiting := make(chan int)
	done := make(chan bool)
	go func() {
		defer func() {
			println("blocked sender saw:", recover().(error).Error())
			done <- true
		}()
		waiting <- 1
	}()
	go func() { close(waiting) }()
	<-done
}

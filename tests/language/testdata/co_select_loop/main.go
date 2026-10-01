package main

// goalchemy:gate cooperative

func main() {
	data := make(chan int)
	quit := make(chan struct{})
	done := make(chan bool)
	go func() {
		total := 0
		for {
			select {
			case v := <-data:
				total += v
				println("got", v)
			case <-quit:
				println("quit, total", total)
				done <- true
				return
			}
		}
	}()
	for i := 1; i <= 3; i++ {
		data <- i * 10
	}
	close(quit)
	<-done
	var nilCh chan int
	ready := make(chan int, 1)
	ready <- 5
	select {
	case v := <-nilCh:
		println("impossible", v)
	case v, ok := <-ready:
		println("ready", v, ok)
	}
	out := make(chan string, 1)
	select {
	case out <- "sent":
		println("send case")
	case <-nilCh:
	}
	println(<-out)
}

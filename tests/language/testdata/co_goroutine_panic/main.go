package main

// goalchemy:gate cooperative

func main() {
	block := make(chan int)
	go func() {
		println("child running")
		panic("child failed")
	}()
	<-block
}

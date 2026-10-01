package main

// goalchemy:gate cooperative

func main() {
	ch := make(chan int)
	println("before")
	<-ch
	println("after")
}

package main

// goalchemy:gate cooperative

func gen(n int) <-chan int {
	out := make(chan int)
	go func() {
		for i := 1; i <= n; i++ {
			out <- i
		}
		close(out)
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- v * v
		}
	}()
	return out
}

func main() {
	sum := 0
	for v := range square(square(gen(5))) {
		println(v)
		sum += v
	}
	println("sum", sum)
}

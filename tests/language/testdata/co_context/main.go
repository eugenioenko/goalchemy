package main

// goalchemy:gate cooperative

import "goalchemy/lib/context"

func worker(ctx context.Context, in <-chan int, out chan<- int) {
	for {
		select {
		case <-ctx.Done():
			println("worker stopped:", ctx.Err().Error())
			close(out)
			return
		case v := <-in:
			out <- v + 1
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	child, cancelChild := context.WithCancel(ctx)
	defer cancelChild()
	in := make(chan int)
	out := make(chan int)
	go worker(child, in, out)
	for i := 0; i < 3; i++ {
		in <- i
		println(<-out)
	}
	println(ctx.Err() == nil, child.Err() == nil)
	cancel()
	for range out {
	}
	println(ctx.Err() == context.Canceled, child.Err() == context.Canceled)
	bg := context.Background()
	println(bg.Err() == nil, bg.Done() == nil)
}

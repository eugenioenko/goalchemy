package main

// goalchemy:gate cooperative

type Source interface{ Next() int }

type chanSource struct{ ch chan int }

func (c chanSource) Next() int { return <-c.ch }

type constSource int

func (c constSource) Next() int { return int(c) }

func drain(s Source, n int) (total int) {
	for i := 0; i < n; i++ {
		total += s.Next()
	}
	return
}

func countdown(ch chan int, n int) int {
	if n == 0 {
		return 0
	}
	ch <- n
	return n + countdown(ch, n-1)
}

func guarded(ch chan string) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = "recovered: " + r.(string)
			ch <- "from defer"
		}
	}()
	ch <- "before panic"
	panic("boom")
}

func main() {
	ch := make(chan int, 10)
	for i := 1; i <= 3; i++ {
		ch <- i * 10
	}
	println(drain(chanSource{ch}, 3), drain(constSource(7), 2))
	var next func() int
	next = constSource(5).Next
	println(next())
	next = chanSource{ch}.Next
	ch <- 42
	println(next())
	steps := make(chan int, 10)
	println(countdown(steps, 4), len(steps))
	msgs := make(chan string)
	done := make(chan string)
	go func() { done <- guarded(msgs) }()
	println(<-msgs)
	println(<-msgs)
	println(<-done)
	type op func(int) int
	ops := []op{
		func(x int) int { return x + 1 },
		func(x int) int { steps <- x; return <-steps * 2 },
	}
	for _, o := range ops {
		println(o(20))
	}
}

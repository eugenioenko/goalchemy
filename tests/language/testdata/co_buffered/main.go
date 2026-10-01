package main

// goalchemy:gate cooperative

func main() {
	ch := make(chan string, 3)
	ch <- "a"
	ch <- "b"
	println(len(ch), cap(ch))
	close(ch)
	for i := 0; i < 4; i++ {
		v, ok := <-ch
		println(i, v, ok)
	}
	var nilc chan int
	println(len(nilc), cap(nilc), nilc == nil)
	type msg struct {
		id   int
		body string
	}
	mc := make(chan msg, 2)
	m := msg{1, "hello"}
	mc <- m
	m.body = "changed"
	got := <-mc
	println(got.id, got.body, m.body)
	cc := make(chan chan int, 1)
	inner := make(chan int, 1)
	cc <- inner
	(<-cc) <- 42
	println(<-inner)
	unbuf := make(chan int)
	go func() { unbuf <- 7 }()
	println(<-unbuf)
}

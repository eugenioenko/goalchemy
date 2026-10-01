package main

// goalchemy:gate cooperative

import (
	"goalchemy/lib/context"
	"goalchemy/lib/time"
)

func main() {
	results := make(chan string)
	for _, w := range []struct {
		name string
		d    time.Duration
	}{{"slow", 60}, {"fast", 20}, {"medium", 40}} {
		go func() {
			time.Sleep(w.d * time.Millisecond)
			results <- w.name
		}()
	}
	for i := 0; i < 3; i++ {
		println(<-results)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	late := make(chan bool)
	go func() {
		time.Sleep(time.Second)
		late <- true
	}()
	select {
	case <-ctx.Done():
		println("timed out:", ctx.Err().Error(), ctx.Err() == context.DeadlineExceeded)
	case <-late:
		println("unexpected")
	}
}

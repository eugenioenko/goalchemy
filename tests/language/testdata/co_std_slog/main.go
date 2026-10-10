package main

// goalchemy:gate cooperative

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	"github.com/eugenioenko/goalchemy/lib/sync"
	"github.com/eugenioenko/goalchemy/std/log/slog"
	"github.com/eugenioenko/goalchemy/std/time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	l := slog.Default().With("job", "rewrap")
	var wg sync.WaitGroup
	results := make(chan int, 3)
	turns := []chan bool{make(chan bool), make(chan bool), make(chan bool)}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			time.Sleep(time.Duration(3-i) * time.Millisecond)
			<-turns[i]
			l.WarnContext(ctx, "segment", "index", i)
			results <- i
			if i > 0 {
				close(turns[i-1])
			}
		}(i)
	}
	close(turns[2])
	wg.Wait()
	cancel()
	close(results)
	for i := range results {
		println("done", i)
	}
	l.ErrorContext(ctx, "after cancel", "err", ctx.Err())
	slog.Log(nil, slog.LevelWarn+1, "nil context")
}

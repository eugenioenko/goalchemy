package main

// goalchemy:gate cooperative

import (
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
)

type label struct{ s string }

func (l label) String() string { return l.s }

type remote struct{ code int }

func (r *remote) Error() string { return fmt.Sprint("remote ", r.code) }

func main() {
	ch := make(chan string, 1)
	ch <- "from channel"
	println(fmt.Sprintf("[%10s]", label{<-ch}))
	err := fmt.Errorf("call: %w", &remote{7})
	var r *remote
	println(errors.As(err, &r), err.Error(), errors.Is(err, err), r.code)
	done := make(chan error)
	go func() { done <- fmt.Errorf("task %d: %w", 1, errors.Join(errors.New("failed"), err)) }()
	got := <-done
	println(got.Error(), errors.As(got, &r))
}

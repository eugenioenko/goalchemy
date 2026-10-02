package rt

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Callback settles only after releasing provider resources. Requests/results
// are snapshotted; first settlement wins and stale owner tokens are discarded.
type Callback func(context.Context, []byte, func([]byte, error)) func()
type Callbacks map[string]Callback

var ErrCallbackUnavailable = errors.New("callback: unavailable")

const callbackMaxBytes = 1024 * 1024

func LibCallbackRequest(t *Task, c Context, name string, request []byte) {
	s := sched
	registry, _ := s.callbacks.(Callbacks)
	cb := registry[name]
	if c.c == nil || (c.c != background && c.c.owner != s) || len(name) == 0 || len(name) > 128 || len(request) > callbackMaxBytes {
		t.RV = []any{[]byte(nil), errors.New("callback: invalid request")}
		return
	}
	if err := StdContextContextErr(c); err != nil {
		t.RV = []any{[]byte(nil), err}
		return
	}
	if cb == nil {
		t.RV = []any{[]byte(nil), ErrCallbackUnavailable}
		return
	}
	if !s.host {
		panic(HostFault{Value: "callback requires host-clock scheduler"})
	}
	ctx, cancel := context.WithCancel(context.Background())
	if c.c.hasDeadline {
		cancel()
		ctx, cancel = context.WithDeadline(context.Background(), s.epoch.Add(time.Duration(c.c.deadline)))
	}
	StdContextContextErr(c)
	if c.c.err != nil {
		cancel()
	}
	// Only the owner touches c, hooks and registration cleanup.
	var token hostToken
	cleanup := func() { cancel(); delete(c.c.hooks, token.id) }
	canceled := func() []any {
		if err := StdContextContextErr(c); err != nil {
			return []any{[]byte(nil), err}
		}
		return nil
	}
	token = s.registerHost(t, cancel, cleanup, canceled)
	if c.c != background && c.c.err == nil {
		if c.c.hooks == nil {
			c.c.hooks = make(map[uint64]func())
		}
		c.c.hooks[token.id] = cancel
	}
	input := append([]byte(nil), request...)
	if request != nil && input == nil {
		input = []byte{}
	}
	s.launchHost(token, func() []any {
		defer cancel()
		type reply struct {
			data []byte
			err  error
		}
		done := make(chan reply, 1)
		var once sync.Once
		settle := func(data []byte, err error) {
			once.Do(func() {
				if len(data) > callbackMaxBytes {
					data = nil
					err = errors.New("callback: reply limit")
				}
				if err != nil {
					data = nil
				}
				copy := append([]byte(nil), data...)
				if data != nil && copy == nil {
					copy = []byte{}
				}
				done <- reply{copy, err}
			})
		}
		stop := cb(ctx, input, settle)
		select {
		case result := <-done:
			return []any{result.data, result.err}
		case <-ctx.Done():
			var stopFault any
			if stop != nil {
				func() { defer func() { stopFault = recover() }(); stop() }()
			}
			<-done
			if stopFault != nil {
				panic(HostFault{Value: stopFault})
			}
			return []any{[]byte(nil), ctx.Err()}
		}
	})
}

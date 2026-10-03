// Package callback declares a bounded asynchronous host callback capability.
package callback

import (
	"context"
	"errors"
	"sync"
)

// MaxBytes is the largest request or reply size (1MiB).
const MaxBytes = 1024 * 1024

// Callback returns a nonblocking cancellation hook. A terminal settle confirms
// provider resources are released. Providers must settle after cancellation.
type Callback func(context.Context, []byte, func([]byte, error)) func()

// Callbacks maps callback names to host providers.
type Callbacks map[string]Callback
type registryKey struct{}

// WithCallbacks returns a child of ctx carrying a copy of callbacks for Request.
func WithCallbacks(ctx context.Context, callbacks Callbacks) context.Context {
	copy := make(Callbacks, len(callbacks))
	for k, v := range callbacks {
		copy[k] = v
	}
	return context.WithValue(ctx, registryKey{}, copy)
}

// Request sends request to the callback registered under name and waits for its reply.
func Request(ctx context.Context, name string, request []byte) ([]byte, error) {
	if ctx == nil || len(name) == 0 || len(name) > 128 || len(request) > MaxBytes {
		return nil, errors.New("callback: invalid request")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	registry, _ := ctx.Value(registryKey{}).(Callbacks)
	cb := registry[name]
	if cb == nil {
		return nil, errors.New("callback: unavailable")
	}
	type reply struct {
		data []byte
		err  error
	}
	done := make(chan reply, 1)
	var once sync.Once
	settle := func(data []byte, err error) {
		once.Do(func() {
			if len(data) > MaxBytes {
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
	input := append([]byte(nil), request...)
	if request != nil && input == nil {
		input = []byte{}
	}
	cancel := cb(ctx, input, settle)
	select {
	case result := <-done:
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return result.data, result.err
	case <-ctx.Done():
		var fault any
		if cancel != nil {
			func() { defer func() { fault = recover() }(); cancel() }()
		}
		<-done
		if fault != nil {
			panic(fault)
		}
		return nil, ctx.Err()
	}
}

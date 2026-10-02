package callback

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestScopedCallbackOwnershipAndRejection(t *testing.T) {
	if _, err := Request(context.Background(), "token", nil); err == nil {
		t.Fatal("missing registry accepted")
	}
	input := []byte{0, 255}
	reply := []byte{255, 0}
	callbacks := Callbacks{"token": func(ctx context.Context, b []byte, settle func([]byte, error)) func() {
		if !bytes.Equal(b, input) {
			t.Fatal("input")
		}
		b[0] = 1
		settle(reply, nil)
		reply[0] = 0
		settle(nil, errors.New("duplicate"))
		return nil
	}}
	ctx := WithCallbacks(context.Background(), callbacks)
	delete(callbacks, "token")
	got, err := Request(ctx, "token", input)
	if err != nil || !bytes.Equal(got, []byte{255, 0}) || input[0] != 0 {
		t.Fatal("ownership/first completion")
	}
	rejected := errors.New("rejected")
	ctx = WithCallbacks(context.Background(), Callbacks{"token": func(_ context.Context, _ []byte, settle func([]byte, error)) func() {
		settle([]byte{1}, rejected)
		return nil
	}})
	got, err = Request(ctx, "token", nil)
	if got != nil || !errors.Is(err, rejected) {
		t.Fatal("provider rejection")
	}
}
func TestCancelAcknowledgment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	stop := make(chan struct{})
	done := make(chan error, 1)
	ctx = WithCallbacks(ctx, Callbacks{"token": func(ctx context.Context, _ []byte, settle func([]byte, error)) func() {
		close(started)
		go func() { <-stop; settle(nil, ctx.Err()) }()
		return func() { close(stop) }
	}})
	go func() { _, err := Request(ctx, "token", nil); done <- err }()
	<-started
	cancel()
	if !errors.Is(<-done, context.Canceled) {
		t.Fatal("cancel")
	}
}

// Package context provides Goalchemy's cancellation contexts; deadlines use
// the virtual clock. This file is the native Go implementation used when
// the program runs with the Go toolchain.
package context

import (
	"context"
	stdtime "time"

	"github.com/eugenioenko/goalchemy/lib/time"
)

// Context carries cancellation across tasks.
type Context = context.Context

// CancelFunc cancels a context.
type CancelFunc = context.CancelFunc

// Canceled is the error of a context canceled by its CancelFunc.
var Canceled = context.Canceled

// DeadlineExceeded is the error of a context whose deadline passed.
var DeadlineExceeded = context.DeadlineExceeded

// Background returns the root context, which is never canceled.
func Background() Context { return context.Background() }

// WithCancel returns a child of parent and a function that cancels it.
func WithCancel(parent Context) (Context, CancelFunc) { return context.WithCancel(parent) }

// WithTimeout returns a child of parent canceled after d.
func WithTimeout(parent Context, d time.Duration) (Context, CancelFunc) {
	return context.WithTimeout(parent, stdtime.Duration(d))
}

// Package runtime provides Goalchemy's scheduler controls. This file is the
// native Go implementation used when the program runs with the Go toolchain.
package runtime

import "runtime"

// Gosched yields to the back of the run queue.
func Gosched() { runtime.Gosched() }

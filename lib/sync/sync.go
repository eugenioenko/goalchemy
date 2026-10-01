// SPDX-License-Identifier: Apache-2.0

// Package sync provides Goalchemy's synchronization primitives. Under
// Goalchemy every target runs one task at a time and these types block
// cooperatively. This file is the native Go implementation used when the
// program runs with the Go toolchain: it aliases the standard types.
package sync

import "sync"

// Mutex is a mutual exclusion lock; waiting tasks acquire it in FIFO order.
type Mutex = sync.Mutex

// WaitGroup waits for a collection of tasks to finish.
type WaitGroup = sync.WaitGroup

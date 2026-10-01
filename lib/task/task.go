// Package task provides structured waiting for Goalchemy programs.
//
// Under Goalchemy every target runs one task at a time, so the functions
// passed to All run in series as cooperative tasks, in order. This file is
// the native Go implementation used when the program runs with the Go
// toolchain; compiled programs use each target's runtime implementation of
// the lib.task.all contract.
package task

import "sync"

// All runs each function as a task and returns when every one has finished.
func All(fns ...func()) {
	var wg sync.WaitGroup
	for _, f := range fns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}

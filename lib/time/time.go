// Package time provides Goalchemy's virtual clock: Sleep advances it
// deterministically. This file is the native Go implementation used when
// the program runs with the Go toolchain.
package time

import "time"

// Duration is a span of virtual time in nanoseconds.
type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

// Sleep suspends the calling task for at least d.
func Sleep(d Duration) { time.Sleep(time.Duration(d)) }

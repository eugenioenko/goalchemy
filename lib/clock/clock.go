// SPDX-License-Identifier: Apache-2.0
// Package clock provides wall time, independent of Goalchemy virtual time.
package clock

import "time"

// Unix returns UTC Unix seconds; it may move backward if the host clock changes.
func Unix() int64 { return time.Now().Unix() }

// UnixNano returns UTC Unix nanoseconds at the host clock's resolution; it may
// move backward if the host clock changes.
func UnixNano() int64 { return time.Now().UnixNano() }

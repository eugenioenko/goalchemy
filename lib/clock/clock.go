// SPDX-License-Identifier: Apache-2.0
// Package clock provides wall time, independent of Goalchemy virtual time.
package clock

import "time"

// Unix returns UTC Unix seconds; it may move backward if the host clock changes.
func Unix() int64 { return time.Now().Unix() }

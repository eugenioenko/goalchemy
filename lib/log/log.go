// SPDX-License-Identifier: Apache-2.0

// Package log routes structured log records from Goalchemy programs to a
// sink the host application installs. Levels follow log/slog: -4 is debug,
// 0 info, 4 warn and 8 error. Without a sink, records at warn and above are
// written to standard error.
package log

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// DefaultLevel is the minimum level delivered before SetHandler is called.
const DefaultLevel = 4

// Record is one log record. Attrs holds key/value pairs with group names
// joined to keys by ".". Text is the record rendered like slog.TextHandler
// without the time attribute.
type Record struct {
	Level   int
	Time    time.Time
	Message string
	Attrs   []string
	Text    string
}

// Handler receives records synchronously; it must not block for long.
type Handler func(Record)

var (
	mu       sync.RWMutex
	handler  Handler
	minLevel = DefaultLevel
)

// SetHandler installs h for records at level and above. A nil h restores the
// standard error sink.
func SetHandler(h Handler, level int) {
	mu.Lock()
	defer mu.Unlock()
	handler, minLevel = h, level
}

// Slog returns a Handler that forwards records to l.
func Slog(l *slog.Logger) Handler {
	return func(r Record) {
		ctx := context.Background()
		if !l.Enabled(ctx, slog.Level(r.Level)) {
			return
		}
		rec := slog.NewRecord(r.Time, slog.Level(r.Level), r.Message, 0)
		for i := 0; i+1 < len(r.Attrs); i += 2 {
			rec.AddAttrs(slog.String(r.Attrs[i], r.Attrs[i+1]))
		}
		_ = l.Handler().Handle(ctx, rec)
	}
}

// Enabled reports whether records at level are delivered.
func Enabled(level int) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level >= minLevel
}

// Emit delivers one record to the installed sink.
func Emit(level int, unixNano int64, message string, attrs []string, text string) {
	mu.RLock()
	h := handler
	mu.RUnlock()
	if h == nil {
		os.Stderr.WriteString(text + "\n")
		return
	}
	func() {
		defer func() { _ = recover() }()
		h(Record{Level: level, Time: time.Unix(0, unixNano).UTC(), Message: message, Attrs: append([]string(nil), attrs...), Text: text})
	}()
}

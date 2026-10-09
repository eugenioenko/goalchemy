package slog

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	"github.com/eugenioenko/goalchemy/std/time"
)

var defaultLogger *Logger

// Default returns the default Logger, which sends records to the host
// application's log sink.
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = New(NewHostHandler(nil))
	}
	return defaultLogger
}

// SetDefault makes l the default Logger.
func SetDefault(l *Logger) { defaultLogger = l }

// A Logger records structured information about each call to its Log,
// Debug, Info, Warn and Error methods and passes it to a Handler.
type Logger struct {
	handler Handler
}

func (l *Logger) clone() *Logger {
	c := *l
	return &c
}

// Handler returns l's Handler.
func (l *Logger) Handler() Handler { return l.handler }

// With returns a Logger that includes the given attributes in each output.
func (l *Logger) With(args ...any) *Logger {
	if len(args) == 0 {
		return l
	}
	c := l.clone()
	c.handler = l.handler.WithAttrs(argsToAttrSlice(args))
	return c
}

// WithGroup returns a Logger that starts a group, if name is non-empty.
func (l *Logger) WithGroup(name string) *Logger {
	if name == "" {
		return l
	}
	c := l.clone()
	c.handler = l.handler.WithGroup(name)
	return c
}

// New creates a new Logger with the given non-nil Handler.
func New(h Handler) *Logger {
	if h == nil {
		panic("nil Handler")
	}
	return &Logger{handler: h}
}

// With calls Logger.With on the default logger.
func With(args ...any) *Logger { return Default().With(args...) }

// Enabled reports whether l emits log records at the given level.
func (l *Logger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	return l.Handler().Enabled(ctx, level)
}

// Log emits a log record with the current time and the given level and
// message. args are alternating keys and values, or Attrs.
func (l *Logger) Log(ctx context.Context, level Level, msg string, args ...any) {
	l.log(ctx, level, msg, args...)
}

// LogAttrs is a more efficient version of Logger.Log that accepts only Attrs.
func (l *Logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	l.logAttrs(ctx, level, msg, attrs...)
}

// Debug logs at LevelDebug.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(context.Background(), LevelDebug, msg, args...)
}

// DebugContext logs at LevelDebug with the given context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelDebug, msg, args...)
}

// Info logs at LevelInfo.
func (l *Logger) Info(msg string, args ...any) { l.log(context.Background(), LevelInfo, msg, args...) }

// InfoContext logs at LevelInfo with the given context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelInfo, msg, args...)
}

// Warn logs at LevelWarn.
func (l *Logger) Warn(msg string, args ...any) { l.log(context.Background(), LevelWarn, msg, args...) }

// WarnContext logs at LevelWarn with the given context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelWarn, msg, args...)
}

// Error logs at LevelError.
func (l *Logger) Error(msg string, args ...any) {
	l.log(context.Background(), LevelError, msg, args...)
}

// ErrorContext logs at LevelError with the given context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, LevelError, msg, args...)
}

func (l *Logger) log(ctx context.Context, level Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	r := NewRecord(time.Now(), level, msg, 0)
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

func (l *Logger) logAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, level) {
		return
	}
	r := NewRecord(time.Now(), level, msg, 0)
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(ctx, r)
}

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) { Default().log(context.Background(), LevelDebug, msg, args...) }

// DebugContext calls Logger.DebugContext on the default logger.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Default().log(ctx, LevelDebug, msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) { Default().log(context.Background(), LevelInfo, msg, args...) }

// InfoContext calls Logger.InfoContext on the default logger.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Default().log(ctx, LevelInfo, msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) { Default().log(context.Background(), LevelWarn, msg, args...) }

// WarnContext calls Logger.WarnContext on the default logger.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Default().log(ctx, LevelWarn, msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, args ...any) { Default().log(context.Background(), LevelError, msg, args...) }

// ErrorContext calls Logger.ErrorContext on the default logger.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Default().log(ctx, LevelError, msg, args...)
}

// Log calls Logger.Log on the default logger.
func Log(ctx context.Context, level Level, msg string, args ...any) {
	Default().log(ctx, level, msg, args...)
}

// LogAttrs calls Logger.LogAttrs on the default logger.
func LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	Default().logAttrs(ctx, level, msg, attrs...)
}

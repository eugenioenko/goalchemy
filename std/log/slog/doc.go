// Package slog provides structured logging with Go's log/slog API. Records
// from the default logger and from NewHostHandler go to a sink the host
// application installs, rendered like slog.TextHandler without the time;
// without a sink, records at LevelWarn and above are written to standard
// error. Values are rendered through std/fmt, source locations are not
// recorded, and there is no TextHandler or JSONHandler.
package slog

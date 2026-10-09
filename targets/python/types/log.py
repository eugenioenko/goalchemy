"""Host log sink for std/log/slog.

Levels follow log/slog: -4 debug, 0 info, 4 warn, 8 error. Without a handler,
records at warn and above are written to standard error.
"""

import datetime
import logging
import threading
from .print import write_stderr

__all__ = ["LogRecord", "set_log_handler", "logging_handler", "log_enabled", "log_emit"]


class LogRecord:
    __slots__ = ("level", "unix_nano", "message", "attrs", "text")

    def __init__(self, level, unix_nano, message, attrs, text):
        self.level = level
        self.unix_nano = unix_nano
        self.message = message
        self.attrs = attrs
        self.text = text

    @property
    def time(self):
        return datetime.datetime.fromtimestamp(self.unix_nano / 1e9, datetime.timezone.utc)

    def __repr__(self):
        return "LogRecord(%r)" % self.text


_lock = threading.Lock()
_handler = None
_level = 4


def set_log_handler(handler, level=4):
    """Route records at level and above to handler(LogRecord); None restores stderr."""
    global _handler, _level
    if type(level) is not int:
        raise TypeError("log level must be an int")
    with _lock:
        _handler, _level = handler, level


def logging_handler(logger=None):
    """A handler that forwards records to a logging.Logger ("goalchemy" by default)."""
    target = logger if logger is not None else logging.getLogger("goalchemy")

    def forward(record):
        level = 20 + record.level * 10 // 4
        if target.isEnabledFor(level):
            target.log(level, "%s", record.text, extra={"goalchemy": record})
    return forward


def log_enabled(level):
    return level >= _level


def _text(b):
    return bytes(b).decode("utf-8", "replace")


def log_emit(level, unix_nano, message, attrs, text):
    handler = _handler
    if handler is None:
        write_stderr(bytes(text) + b"\n")
        return
    pairs = [(_text(attrs[i]), _text(attrs[i + 1])) for i in range(0, len(attrs) - 1, 2)]
    try:
        handler(LogRecord(level, unix_nano, _text(message), pairs, _text(text)))
    except Exception:
        pass

"""std.context.err and the context representation: a cancellation state with
a Done channel closed on cancellation; Background's channel is nil."""

from .chan_close import chan_close
from .chan_make import make_chan
from .std_errors_new import std_errors_new


class Context:
    __slots__ = ("done", "err", "children", "__weakref__")

    def __init__(self, done):
        self.done = done
        self.err = None
        self.children = []


CONTEXT_CANCELED = std_errors_new(b"context canceled")
CONTEXT_DEADLINE_EXCEEDED = std_errors_new(b"context deadline exceeded")
BACKGROUND = Context(None)


def cancel_context(c, err):
    if c.err is not None:
        return
    c.err = err
    chan_close(c.done)
    for k in c.children:
        cancel_context(k, err)
    c.children = []


def new_child(parent):
    c = Context(make_chan(0, lambda: None))
    if parent.err is not None:
        cancel_context(c, parent.err)
    elif parent is not BACKGROUND:
        parent.children.append(c)
    return c


def std_context_context_err(c):
    return c.err

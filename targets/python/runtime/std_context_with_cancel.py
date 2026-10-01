"""std.context.with_cancel: a child cancelled by its cancel function."""

from .std_context_err import CONTEXT_CANCELED, cancel_context, new_child


def std_context_with_cancel(parent):
    c = new_child(parent)
    return c, lambda: cancel_context(c, CONTEXT_CANCELED)

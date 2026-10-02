"""std.context.with_cancel: inherit and register the earliest deadline."""
from .std_context_err import CONTEXT_CANCELED, cancel_context, new_child, arm_deadline


def std_context_with_cancel(parent):
    c = new_child(parent)
    arm_deadline(c)
    return c, lambda: cancel_context(c, CONTEXT_CANCELED)

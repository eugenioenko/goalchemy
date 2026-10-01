"""std.context.with_timeout: a child cancelled after a virtual-time duration."""

from .std_context_err import CONTEXT_CANCELED, CONTEXT_DEADLINE_EXCEEDED, cancel_context, new_child
from .task_spawn import sched


def std_context_with_timeout(parent, d):
    c = new_child(parent)
    if c.err is None:
        sched().add_timer(d, None, lambda: cancel_context(c, CONTEXT_DEADLINE_EXCEEDED))
    return c, lambda: cancel_context(c, CONTEXT_CANCELED)

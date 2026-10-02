"""std.context.done: owner-checked cancellation channel."""
from .std_context_err import check_context


def std_context_context_done(c):
    return check_context(c).done

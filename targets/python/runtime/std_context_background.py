"""std.context.background: the never-cancelled root context."""

from .std_context_err import BACKGROUND


def std_context_background():
    return BACKGROUND

"""std.context.with_timeout: captured absolute monotonic/virtual deadline."""
from .std_context_err import CONTEXT_CANCELED, cancel_context, host_boundary


def std_context_with_timeout(parent, d):
    c = host_boundary(parent, d)
    return c, lambda: cancel_context(c, CONTEXT_CANCELED)

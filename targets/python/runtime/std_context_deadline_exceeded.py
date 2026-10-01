"""std.context.deadline_exceeded: the context.DeadlineExceeded sentinel."""

from .std_context_err import CONTEXT_DEADLINE_EXCEEDED


def std_context_deadline_exceeded():
    return CONTEXT_DEADLINE_EXCEEDED

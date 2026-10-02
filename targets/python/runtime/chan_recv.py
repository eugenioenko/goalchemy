"""core.chan.recv: v, ok := <-ch, a pause primitive; results arrive in
t.rv as [value, ok]."""

from .chan_make import attach_waiter, Waiter, try_recv
from .task_spawn import sched


def chan_recv(t, ch):
    sched().check()
    if ch is None:
        sched().block(t)
        return
    v, ok, done = try_recv(ch)
    if done:
        t.rv = [v, ok]
        return
    attach_waiter(t, ch.recvq, Waiter(t, None))
    sched().block(t)

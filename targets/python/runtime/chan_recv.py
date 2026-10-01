"""core.chan.recv: v, ok := <-ch, a pause primitive; results arrive in
t.rv as [value, ok]."""

from .chan_make import Waiter, try_recv
from .task_spawn import sched


def chan_recv(t, ch):
    if ch is None:
        sched().block(t)
        return
    v, ok, done = try_recv(ch)
    if done:
        t.rv = [v, ok]
        return
    ch.recvq.append(Waiter(t, None))
    sched().block(t)

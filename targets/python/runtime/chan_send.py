"""core.chan.send: ch <- v, a pause primitive."""

from ..types.panic import plain_panic
from .chan_make import Waiter, dequeue
from .task_spawn import sched


def chan_send(t, ch, v):
    t.rv = []
    if ch is None:
        sched().block(t)
        return
    if ch.closed:
        raise plain_panic("send on closed channel")
    r = dequeue(ch.recvq)
    if r is not None:
        r.recv_done(v, True)
        return
    if len(ch.buf) < ch.size:
        ch.buf.append(v)
        return
    ch.sendq.append(Waiter(t, v))
    sched().block(t)

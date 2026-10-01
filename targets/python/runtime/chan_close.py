"""core.chan.close: close(ch), releasing waiting receivers and senders."""

from ..types.panic import plain_panic
from .chan_make import dequeue


def chan_close(ch):
    if ch is None:
        raise plain_panic("close of nil channel")
    if ch.closed:
        raise plain_panic("close of closed channel")
    ch.closed = True
    w = dequeue(ch.recvq)
    while w is not None:
        w.recv_done(ch.zero(), False)
        w = dequeue(ch.recvq)
    w = dequeue(ch.sendq)
    while w is not None:
        w.send_done(True)
        w = dequeue(ch.sendq)

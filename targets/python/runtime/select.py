"""core.select: commit exactly one ready case; a pause primitive whose
results arrive in t.rv as [index, value, ok], index -1 for the default.
Each case is (channel, is_send, value)."""

from ..types.panic import plain_panic
from .chan_make import SelectState, Waiter, dequeue, has_live, try_recv
from .task_spawn import sched


def select(t, cases, has_default):
    ready = []
    for i, (ch, send, _) in enumerate(cases):
        if ch is None:
            continue
        if send:
            if ch.closed or has_live(ch.recvq) or len(ch.buf) < ch.size:
                ready.append(i)
        elif ch.buf or has_live(ch.sendq) or ch.closed:
            ready.append(i)
    if ready:
        i = ready[sched().choose(len(ready))]
        ch, send, val = cases[i]
        if send:
            if ch.closed:
                raise plain_panic("send on closed channel")
            w = dequeue(ch.recvq)
            if w is not None:
                w.recv_done(val, True)
            else:
                ch.buf.append(val)
            t.rv = [i, None, False]
            return
        v, ok, _ = try_recv(ch)
        t.rv = [i, v, ok]
        return
    if has_default:
        t.rv = [-1, None, False]
        return
    st = SelectState()
    for i, (ch, send, val) in enumerate(cases):
        if ch is None:
            continue
        w = Waiter(t, val if send else None, st, i)
        (ch.sendq if send else ch.recvq).append(w)
    sched().block(t)

    def cleanup():
        for ch, _, _ in cases:
            if ch is not None:
                ch.sendq = [w for w in ch.sendq if w.sel is not st]
                ch.recvq = [w for w in ch.recvq if w.sel is not st]

    t.cleanup = cleanup

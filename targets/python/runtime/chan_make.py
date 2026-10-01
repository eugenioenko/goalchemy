"""core.chan.make: make(chan T, size), and the channel state shared by the
channel and select runtime functions."""

from ..types.panic import plain_panic
from .task_spawn import sched


class SelectState:
    __slots__ = ("done",)

    def __init__(self):
        self.done = False


class Waiter:
    __slots__ = ("task", "val", "sel", "idx")

    def __init__(self, task, val, sel=None, idx=0):
        self.task = task
        self.val = val
        self.sel = sel
        self.idx = idx

    def recv_done(self, val, ok):
        """Completes a waiting receiver with a value or closure."""
        if self.sel is not None:
            self.sel.done = True
            self.task.rv = [self.idx, val, ok]
        else:
            self.task.rv = [val, ok]
        sched().ready(self.task)

    def send_done(self, closed):
        """Completes a waiting sender; closed makes it panic when it resumes."""
        if self.sel is not None:
            self.sel.done = True
            self.task.rv = [self.idx, None, False]
        else:
            self.task.rv = []
        if closed:
            self.task.resume_panic = plain_panic("send on closed channel")
        sched().ready(self.task)


class Chan:
    __slots__ = ("buf", "size", "closed", "recvq", "sendq", "zero", "__weakref__")

    def __init__(self, size, zero):
        self.buf = []
        self.size = size
        self.closed = False
        self.recvq = []
        self.sendq = []
        self.zero = zero


def dequeue(q):
    while q:
        w = q.pop(0)
        if w.sel is None or not w.sel.done:
            return w
    return None


def has_live(q):
    return any(w.sel is None or not w.sel.done for w in q)


def try_recv(ch):
    """Receives without blocking: (value, ok, done)."""
    if ch.buf:
        v = ch.buf.pop(0)
        w = dequeue(ch.sendq)
        if w is not None:
            ch.buf.append(w.val)
            w.send_done(False)
        return v, True, True
    w = dequeue(ch.sendq)
    if w is not None:
        v = w.val
        w.send_done(False)
        return v, True, True
    if ch.closed:
        return ch.zero(), False, True
    return None, False, False


def make_chan(size, zero=lambda: None):
    if size < 0 or size > 2**53:
        raise plain_panic("makechan: size out of range")
    return Chan(size, zero)

"""std.sync.mutex.lock: acquire or wait FIFO; Unlock hands the lock over."""

from .task_spawn import sched


class Mutex:
    __slots__ = ("locked", "waiters", "__weakref__")

    def __init__(self):
        self.locked = False
        self.waiters = []

    def _clone(self):
        m = Mutex()
        m._set(self)
        return m

    def _set(self, o):
        self.locked = o.locked
        self.waiters = list(o.waiters)


def std_sync_mutex_lock(t, m):
    sched().check()
    """A pause primitive."""
    t.rv = []
    if not m.locked:
        m.locked = True
        return
    m.waiters.append(t)
    def cleanup():
        if t in m.waiters:
            m.waiters.remove(t)
    t.cleanup = cleanup
    sched().block(t)

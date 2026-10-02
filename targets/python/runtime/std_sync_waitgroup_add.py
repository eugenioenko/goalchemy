"""std.sync.waitgroup.add: adjust the counter; release waiters at zero."""

from ..types.iface import box
from ..types.panic import GoPanic
from ..types.program import STRING_TYPE
from .task_spawn import sched


class WaitGroup:
    __slots__ = ("n", "waiters", "__weakref__")

    def __init__(self):
        self.n = 0
        self.waiters = []

    def _clone(self):
        w = WaitGroup()
        w._set(self)
        return w

    def _set(self, o):
        self.n = o.n
        self.waiters = list(o.waiters)


def std_sync_waitgroup_add(wg, d):
    from .task_spawn import sched
    sched().check()
    wg.n += d
    if wg.n < 0:
        raise GoPanic(box(STRING_TYPE, b"sync: negative WaitGroup counter"))
    if wg.n == 0:
        for t in wg.waiters:
            sched().ready(t)
        wg.waiters = []

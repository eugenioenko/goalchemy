"""std.sync.waitgroup.wait: return at zero, otherwise wait; a pause primitive."""

from .task_spawn import sched


def std_sync_waitgroup_wait(t, wg):
    t.rv = []
    if wg.n == 0:
        return
    wg.waiters.append(t)
    sched().block(t)

"""std.sync.waitgroup.wait: return at zero, otherwise wait; a pause primitive."""

from .task_spawn import sched


def std_sync_waitgroup_wait(t, wg):
    sched().check()
    t.rv = []
    if wg.n == 0:
        return
    wg.waiters.append(t)
    def cleanup():
        if t in wg.waiters:
            wg.waiters.remove(t)
    t.cleanup = cleanup
    sched().block(t)

"""std.sync.mutex.unlock: hand the lock to the first waiter or release it."""

from .task_spawn import fatal, sched


def std_sync_mutex_unlock(m):
    if not m.locked:
        fatal("sync: unlock of unlocked mutex")
    if m.waiters:
        sched().ready(m.waiters.pop(0))
        return
    m.locked = False

"""std.time.sleep: suspend for d nanoseconds of virtual time; a pause primitive."""

from .task_spawn import sched, yield_task


def std_time_sleep(t, d):
    t.rv = []
    if d <= 0:
        yield_task(t)
        return
    owner = sched()
    timer = owner.add_timer(d, t, None)
    t.cleanup = lambda: owner.remove_timer(timer)
    sched().block(t)

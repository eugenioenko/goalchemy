"""std.runtime.gosched: requeue the running task; a pause primitive."""

from .task_spawn import yield_task


def std_runtime_gosched(t):
    t.rv = []
    yield_task(t)

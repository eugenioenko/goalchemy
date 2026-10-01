"""lib.task.all: run each function as a task, in order, and resume the
caller when all have finished; a pause primitive. The functions arrive in
the resumable form (each call returns a frame)."""

from ..types.panic import nil_deref
from .task_spawn import Frame, call, ret, sched, spawn


class _AllChild(Frame):
    def __init__(self, f, st):
        super().__init__()
        self.f = f
        self.st = st

    def step(self, t):
        if self.pc == 0:
            self.pc = 1
            if self.f is None:
                raise nil_deref()
            call(t, self.f())
            return
        self.st[0] -= 1
        if self.st[0] == 0:
            sched().ready(self.st[1])
        ret(t, self)


def lib_task_all(t, fns):
    t.rv = []
    if fns.l == 0:
        return
    st = [fns.l, t]
    for i in range(fns.l):
        spawn(_AllChild(fns.a[fns.o + i], st))
    sched().block(t)

"""core.task.spawn and the cooperative scheduler.

Suspending functions are compiled to resumable frames: a frame holds the
function's locals and the block to resume at, and step runs it until it
returns or reaches a pause point. A task is a stack of frames driven by a
trampoline; exactly one task runs at a time and runnable tasks are
dispatched in FIFO order. Pause primitives either complete immediately,
leaving their results in task.rv, or block the task until another task or
a timer readies it. Deferred calls, panics, and recover are managed per task
by the runtime.
"""

import os

from ..types.panic import GoPanic, nil_deref
from ..types.program import catch_panic, format_chain, panic_state, report_panic, _run_large_stack
from ..types.print import write_stderr
from ..types.slice import Slice


class Frame:
    def __init__(self):
        self.pc = 0
        self.defers = []
        self.parent = None
        self.panicking = None

    def step(self, t):
        raise NotImplementedError

    def results(self):
        return []


class Task:
    def __init__(self, tid, frame):
        self.id = tid
        self.frame = frame
        self.rv = []
        self.blocked = False
        self.done = False
        self.resume_panic = None
        self.cur_panic = None
        self.defer_target = None
        self.cleanup = None


class Blocked(Exception):
    """Raised out of a harness case when its task blocks with nothing runnable."""


class _FatalPanic(Exception):
    def __init__(self, p):
        super().__init__("fatal panic")
        self.p = p


def _seed():
    try:
        s = int(os.environ.get("GOALCHEMY_SEED", "1"))
    except ValueError:
        return 1
    return s if 0 < s < 2**32 else 1


def fatal(msg):
    write_stderr(b"fatal error: " + msg.encode() + b"\n")
    os._exit(2)


class Scheduler:
    def __init__(self, main):
        self.runq = []
        self.cur = main
        self.main = main
        self.next_id = 1
        self.rng = _seed()
        self.clock = 0
        self.timers = []
        self.seq = 0
        self.harness = False

    def choose(self, n):
        """xorshift32 choice source, identical on every target."""
        x = self.rng
        x ^= (x << 13) & 0xFFFFFFFF
        x ^= x >> 17
        x ^= (x << 5) & 0xFFFFFFFF
        self.rng = x
        return x % n

    def ready(self, t):
        self.runq.append(t)

    def block(self, t):
        t.blocked = True

    def add_timer(self, d, task, fn):
        self.seq += 1
        self.timers.append((self.clock + d, self.seq, task, fn))

    def next(self):
        while not self.runq:
            if not self.timers:
                if self.harness:
                    raise Blocked()
                fatal("all goroutines are asleep - deadlock!")
            self.fire_timers()
        return self.runq.pop(0)

    def fire_timers(self):
        at = min(t[0] for t in self.timers)
        self.clock = at
        due = sorted((t for t in self.timers if t[0] == at), key=lambda t: t[1])
        self.timers = [t for t in self.timers if t[0] != at]
        for _, _, task, fn in due:
            if fn is not None:
                fn()
            if task is not None:
                self.ready(task)

    def run(self, t):
        """Drives a task's frames until it blocks or finishes."""
        self.cur = t
        t.blocked = False
        if t.cleanup is not None:
            c, t.cleanup = t.cleanup, None
            c()
        while not t.blocked and t.frame is not None:
            p = t.resume_panic
            if p is not None:
                t.resume_panic = None
                self.exit(t, t.frame, p)
                continue
            f = t.frame
            try:
                f.step(t)
            except GoPanic as e:
                self.exit(t, f, e)

    def exit(self, t, f, p):
        """Leaves frame f (panicking when p is set); deferred calls run first."""
        if p is not None:
            if p.prev is None and f.panicking is not None and f.panicking is not p:
                p.prev = f.panicking
            f.panicking = p
        t.frame = f
        if f.defers:
            r = _DeferRunner(f)
            r.parent = f
            t.frame = r
            return
        self.finish(t, f)

    def finish(self, t, f):
        """Pops f and delivers its results, or its panic, to the caller."""
        p = f.panicking
        parent = f.parent
        t.frame = parent
        if parent is None:
            t.done = True
            if p is not None:
                if self.harness:
                    raise _FatalPanic(p)
                report_panic(p)
            return
        if isinstance(parent, _DeferRunner) and parent.child is f:
            parent.child_done(t, p)
            return
        if p is not None:
            self.exit(t, parent, p)
            return
        t.rv = f.results()


class _DeferRunner(Frame):
    """Runs a frame's deferred calls in reverse order."""

    def __init__(self, target):
        super().__init__()
        self.target = target
        self.child = None
        self.saved = (None, None)

    def step(self, t):
        tf = self.target
        while tf.defers:
            d = tf.defers.pop()
            self.saved = (t.cur_panic, t.defer_target)
            t.cur_panic, t.defer_target = tf.panicking, d.fid
            if d.start:
                try:
                    if d.f is None:
                        raise nil_deref()
                    child = d.f(*d.args)
                except GoPanic as e:
                    t.cur_panic, t.defer_target = self.saved
                    self.after(tf, e)
                    continue
                self.child = child
                child.parent = self
                t.frame = child
                return
            p = None
            try:
                if d.f is None:
                    raise nil_deref()
                d.f(*d.args)
            except GoPanic as e:
                p = e
            t.cur_panic, t.defer_target = self.saved
            self.after(tf, p)
        t.frame = tf
        sched().finish(t, tf)

    def child_done(self, t, p):
        t.cur_panic, t.defer_target = self.saved
        self.child = None
        t.frame = self
        self.after(self.target, p)

    def after(self, tf, p):
        if p is not None:
            if p.prev is None and tf.panicking is not None and tf.panicking is not p:
                p.prev = tf.panicking
            tf.panicking = p
            return
        if tf.panicking is not None and tf.panicking.recovered:
            tf.panicking = None


_sched = [Scheduler(Task(0, None))]


def sched():
    return _sched[0]


def call(t, child):
    """Pushes a callee frame; the caller resumes with its results in t.rv."""
    child.parent = t.frame
    t.frame = child


def ret(t, f):
    sched().exit(t, f, None)


class _SyncFrame(Frame):
    def __init__(self, fn):
        super().__init__()
        self.fn = fn
        self.res = []

    def step(self, t):
        self.res = self.fn()
        ret(t, self)

    def results(self):
        return self.res


def sync(fn):
    """Runs an ordinary call as a frame."""
    return _SyncFrame(fn)


def adapt(f, n):
    """Adapts an ordinary function value with n results to the resumable form."""
    if f is None:
        return None

    def g(*a):
        def run():
            r = f(*a)
            return [] if n == 0 else [r] if n == 1 else list(r)

        return sync(run)

    g._fid = getattr(f, "_fid", None)
    return g


def adapt_slice(s, n):
    if s.a is None:
        return s
    a = [adapt(s.a[s.o + i], n) for i in range(s.l)]
    return Slice(a, 0, len(a), len(a))


def spawn(f):
    """go f(args): starts a task running frame f."""
    s = sched()
    t = Task(s.next_id, f)
    s.next_id += 1
    s.ready(t)


def _install(main, harness):
    s = Scheduler(main)
    s.harness = harness
    _sched[0] = s
    panic_state.get = lambda: _sched[0].cur
    return s


def run_main(entry):
    """Runs the program entry as the first task until it returns."""

    def body():
        main = Task(0, entry)
        s = _install(main, False)
        s.ready(main)
        while not main.done:
            s.run(s.next())

    _run_large_stack(body)


def yield_task(t):
    """Requeues the running task: a pause primitive."""
    sched().ready(t)
    sched().block(t)


class _AwaitFrame(Frame):
    def __init__(self, fn):
        super().__init__()
        self.fn = fn
        self.res = []

    def step(self, t):
        if self.pc == 0:
            self.pc = 1
            self.fn(t)
            return
        self.res = t.rv
        ret(t, self)

    def results(self):
        return self.res


def run_isolated(fn):
    """Runs one pause primitive in an isolated scheduler for a harness case;
    raises Blocked when no task can run, and the source panic on panic."""
    h = _AwaitFrame(fn)
    main = Task(0, h)
    s = _install(main, True)
    s.ready(main)
    try:
        while not main.done:
            s.run(s.next())
    except _FatalPanic as e:
        raise e.p
    return h.res


def reset_scheduler():
    _install(Task(0, None), True)

"""Program-level runtime: deferred calls, recover, closures, and the entry
point that reports unrecovered panics as the Go runtime does."""

import os
import sys
import threading

from .iface import Box, box, type_desc
from .panic import GoPanic, TYPE_ASSERTION_ERROR, nil_deref
from .print import write_stderr


class HostFault(Exception):
    """Unexpected native bridge fault, outside source panic/recover."""


class SourceFatal(BaseException):
    """Unrecoverable source runtime fatal, reported only after owner cleanup."""
    def __init__(self, report):
        self.report = report


class Deferred:
    """A deferred call with its callee and arguments evaluated. start is set
    when calling f returns a frame (a suspending callee)."""

    __slots__ = ("f", "args", "fid", "start")

    def __init__(self, f, args, fid, start=False):
        self.f = f
        self.args = args
        self.fid = fid
        self.start = start


class PanicState:
    __slots__ = ("cur_panic", "defer_target")

    def __init__(self):
        self.cur_panic = None
        self.defer_target = None


_main_state = PanicState()


class _Current:
    """Returns the running task's recover state; the scheduler replaces it."""

    def __init__(self):
        self.get = lambda: _main_state


panic_state = _Current()

PANIC_NIL_ERROR = type_desc(
    name="*runtime.PanicNilError",
    kind="runtime_error",
    eq=lambda a, b: a is b,
    key=lambda a: id(a),
    methods={"Error": lambda v: b"runtime error: panic called with nil argument", "RuntimeError": lambda v: None},
)

STRING_TYPE = type_desc(name="string", kind="string", eq=lambda a, b: a == b, key=lambda a: a, methods={}, basic="string")


def go_panic(v):
    """panic(v): a nil interface becomes *runtime.PanicNilError."""
    return GoPanic(box(PANIC_NIL_ERROR, object()) if v is None else v)


def catch_panic(e):
    if isinstance(e, GoPanic):
        return e
    raise e


def run_defers(ds, p):
    """Runs deferred calls in reverse order, implementing recover."""
    panicking = p
    while ds:
        d = ds.pop()
        st = panic_state.get()
        saved_panic, saved_target = st.cur_panic, st.defer_target
        st.cur_panic, st.defer_target = panicking, d.fid
        try:
            if d.f is None:
                raise nil_deref()
            d.f(*d.args)
        except GoPanic as np:
            if np is not panicking and np.prev is None:
                np.prev = panicking
            panicking = np
            continue
        finally:
            st.cur_panic, st.defer_target = saved_panic, saved_target
        if panicking is not None and panicking.recovered:
            panicking = None
    if panicking is not None:
        raise panicking


def recover(fid):
    st = panic_state.get()
    p = st.cur_panic
    if p is None or p.recovered or st.defer_target != fid:
        return None
    p.recovered = True
    return p.value


def closure(fid, f):
    """Tags a function value with the identity recover compares against."""
    f._fid = fid
    return f


def bound(fid, f, recv):
    return closure(fid, lambda *a: f(recv, *a))


def ichk(x):
    if x is None:
        raise nil_deref()
    return x


def ibound(x, mid):
    b = ichk(x)
    m = b.t.methods[mid]
    return closure(getattr(m, "_fid", None), lambda *a: m(b.v, *a))


def fnchk(f):
    if f is None:
        raise nil_deref()
    return f


def fid(f):
    return None if f is None else getattr(f, "_fid", None)


def assert_panic(x, iface, target, missing):
    if x is None:
        msg = "interface conversion: " + iface + " is nil, not " + target
    elif missing is not None:
        msg = "interface conversion: " + x.t.name + " is not " + target + ": missing method " + missing
    else:
        msg = "interface conversion: " + iface + " is " + x.t.name + ", not " + target
    return GoPanic(box(TYPE_ASSERTION_ERROR, msg.encode("latin-1")))


def _indented(b):
    return b.replace(b"\n", b"\n\t")


def format_panic_value(v):
    if v is None:
        return b"nil"
    m = v.t.methods
    if "Error" in m:
        return _indented(m["Error"](v.v))
    if "String" in m:
        return _indented(m["String"](v.v))
    name = v.t.name.encode("latin-1")
    builtin = b"." not in name
    if v.t.basic == "string":
        return _indented(v.v) if builtin else name + b'("' + _indented(v.v) + b'")'
    if v.t.basic == "bool":
        s = b"true" if v.v else b"false"
        return s if builtin else name + b"(" + s + b")"
    if v.t.basic == "int":
        s = str(v.v).encode()
        return s if builtin else name + b"(" + s + b")"
    return b"(" + name + b") 0xc000000000"


def format_chain(p):
    s = b""
    if p.prev is not None:
        s += format_chain(p.prev) + b"\t"
    s += b"panic: " + format_panic_value(p.value)
    if p.recovered:
        s += b" [recovered]"
    return s + b"\n"


def report_panic(p):
    write_stderr(format_chain(p))
    sys.stderr.flush()
    os._exit(2)


def _run_large_stack(fn):
    """Runs fn on a thread with a large stack so deep recursion behaves."""
    sys.setrecursionlimit(200000)
    threading.stack_size(512 * 1024 * 1024)
    result = []

    def target():
        try:
            fn()
        except RecursionError:
            write_stderr(b"runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n")
            os._exit(2)
        except GoPanic as p:
            report_panic(p)
        except BaseException as e:  # noqa: BLE001 - faults are reported and exit
            if isinstance(e, SourceFatal):
                write_stderr(e.report)
                os._exit(2)
            result.append(e)

    t = threading.Thread(target=target)
    t.start()
    t.join()
    if result:
        raise result[0]


_entry_lock = threading.Lock()
_entry_active = False


def reserve():
    global _entry_active
    with _entry_lock:
        if _entry_active:
            raise HostFault("overlapping executable entry/reset")
        _entry_active = True


def unreserve():
    global _entry_active
    with _entry_lock:
        _entry_active = False


sequential_start = lambda: None
sequential_end = lambda: None


def main(entry):
    """Legacy sequential executable, serialized with virtual/host/harness entries."""
    reserve()
    def drive():
        sequential_start()
        try:
            entry()
        finally:
            sequential_end()
    try:
        _run_large_stack(drive)
        sys.stdout.flush()
    finally:
        unreserve()


# This guard is enabled only by the explicit host owner. It never changes the
# process recursion limit or default native thread stack size. The legacy virtual
# executable keeps its historical conformance policy.
source_guard_state = threading.local()


def source_guard(fn):
    def guarded(*args, **kwargs):
        if not getattr(source_guard_state, "enabled", False):
            return fn(*args, **kwargs)
        depth = getattr(source_guard_state, "depth", 0)
        if depth >= 128:
            raise SourceFatal(b"runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n")
        source_guard_state.depth = depth + 1
        try:
            return fn(*args, **kwargs)
        finally:
            source_guard_state.depth = depth
    return guarded

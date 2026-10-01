"""Panic and runtime error representation for the Python target."""

from .iface import Box, TypeDesc, box, type_desc


class GoPanic(Exception):
    """A source panic carrying a Go interface value (a Box or None)."""

    def __init__(self, value):
        super().__init__("go panic")
        self.value = value
        self.recovered = False
        self.prev = None


class Fault(Exception):
    """An implementation fault: never a source panic."""


def fault(msg):
    return Fault("goalchemy fault: " + msg)


def _error_type(name, prefix):
    return type_desc(
        name=name,
        kind="runtime_error",
        eq=lambda a, b: a == b,
        key=lambda a: a,
        methods={"Error": lambda msg: prefix + msg, "RuntimeError": lambda msg: None},
    )


RUNTIME_ERROR = _error_type("runtime.Error", b"runtime error: ")
PLAIN_ERROR = _error_type("runtime.plainError", b"")
TYPE_ASSERTION_ERROR = _error_type("*runtime.TypeAssertionError", b"")


def runtime_panic(msg):
    return GoPanic(box(RUNTIME_ERROR, msg.encode("latin-1") if isinstance(msg, str) else msg))


def plain_panic(msg):
    return GoPanic(box(PLAIN_ERROR, msg.encode("latin-1") if isinstance(msg, str) else msg))


def uncomparable(name):
    return runtime_panic("comparing uncomparable type " + name)


def unhashable(name):
    return runtime_panic("hash of unhashable type " + name)


def uncomparable_eq(name):
    raise uncomparable(name)


def unhashable_key(name):
    raise unhashable(name)


def nil_deref():
    return runtime_panic("invalid memory address or nil pointer dereference")


def nilchk(p):
    if p is None:
        raise nil_deref()
    return p


def index_panic(i, n):
    if i < 0:
        return runtime_panic("index out of range [%d]" % i)
    return runtime_panic("index out of range [%d] with length %d" % (i, n))


def idx(i, n):
    """Checks an index against a length and returns it."""
    if i < 0 or i >= n:
        raise index_panic(i, n)
    return i

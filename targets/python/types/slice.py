"""Value-like slice headers over lists or native one-byte bytearray storage."""

from .panic import fault

HOST_LIMIT = 2**32 - 1


class Slice:
    __slots__ = ("a", "o", "l", "c", "b")

    def __init__(self, a, o, l, c, byte=False):
        self.a = a
        self.o = o
        self.l = l
        self.c = c
        self.b = byte or isinstance(a, bytearray)


NIL = Slice(None, 0, 0, 0)
BYTE_NIL = Slice(None, 0, 0, 0, True)


def alloc_bytes(c):
    if c > HOST_LIMIT:
        raise fault("allocation of %d elements exceeds host limits" % c)
    try:
        return bytearray(c)
    except (MemoryError, OverflowError):
        raise fault("allocation of %d elements exceeds available host memory" % c)


def from_list(a):
    return Slice(a, 0, len(a), len(a))

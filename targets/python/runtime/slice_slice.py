"""core.slice.slice: s[lo:hi:max] sharing backing storage."""

from ..types.slice import NIL, Slice
from ..types.bounds import check2, check3


def reslice(s, lo=None, hi=None, mx=None):
    lo = 0 if lo is None else lo
    hi = s.l if hi is None else hi
    if mx is not None:
        check3(lo, hi, mx, s.c, "capacity")
    else:
        check2(lo, hi, s.c, "capacity")
        mx = s.c
    if s.a is None:
        return s
    return Slice(s.a, s.o + lo, hi - lo, mx - lo, s.b)


def slice_array(a, lo=None, hi=None, mx=None):
    """Slices an array through a pointer: (&a)[lo:hi:max]."""
    lo = 0 if lo is None else lo
    hi = len(a) if hi is None else hi
    if mx is not None:
        check3(lo, hi, mx, len(a), "length")
    else:
        check2(lo, hi, len(a), "length")
        mx = len(a)
    return Slice(a, lo, hi - lo, mx - lo)

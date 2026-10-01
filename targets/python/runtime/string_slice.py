"""core.string.slice: substring by byte bounds."""

from ..types.bounds import check2


def sslice(s, lo=None, hi=None):
    lo = 0 if lo is None else lo
    hi = len(s) if hi is None else hi
    check2(lo, hi, len(s), "length")
    return s[lo:hi]

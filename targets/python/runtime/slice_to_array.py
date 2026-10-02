"""core.slice.to_array: [N]T(s), copying the first N elements."""

from ..types.panic import runtime_panic


def slice_to_array(s, n, clone=None):
    if s.l < n:
        raise runtime_panic("cannot convert slice with length %d to array or pointer to array with length %d" % (s.l, n))
    if s.b:
        return bytearray() if n == 0 else s.a[s.o:s.o + n]
    if n == 0:
        return []
    return [(clone(v) if clone else v) for v in s.a[s.o:s.o + n]]

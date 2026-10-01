"""core.slice.append: append with Goalchemy's growth rule. Aggregate
elements are copied with clone when they move to new storage."""

from ..types.slice import NIL, Slice
from ..types.panic import fault

MAX_CAP = 2**63 - 1


def grow_cap(old, required):
    doubled = 2 * old if old <= MAX_CAP // 2 else MAX_CAP
    return max(required, max(1, doubled))


def _append_values(s, vs, clone=None):
    n = s.l + len(vs)
    if not vs:
        return s
    if n <= s.c:
        a = s.a
        for i, v in enumerate(vs):
            a[s.o + s.l + i] = v
        return Slice(a, s.o, n, s.c)
    c = grow_cap(s.c, n)
    if c > 2**32 - 1:
        raise fault("slice growth to %d elements exceeds host limits" % c)
    a = [None] * c
    for i in range(s.l):
        v = s.a[s.o + i]
        a[i] = clone(v) if clone else v
    for i, v in enumerate(vs):
        a[s.l + i] = v
    return Slice(a, 0, n, c)


def append(s, vs, clone=None):
    return _append_values(s, vs, clone)


def append_slice(s, t, clone=None):
    if t.l == 0:
        return NIL if s.a is None and t.a is None else s
    vs = [(clone(v) if clone else v) for v in t.a[t.o:t.o + t.l]]
    return _append_values(s, vs, clone)


def append_string(b, s):
    return _append_values(b, list(s))

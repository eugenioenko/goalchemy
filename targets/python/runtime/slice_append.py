"""core.slice.append: append with Goalchemy's growth rule. Aggregate
elements are copied with clone when they move to new storage."""

from ..types.slice import Slice, alloc_bytes
from ..types.panic import fault

MAX_CAP = 2**63 - 1


def grow_cap(old, required):
    doubled = 2 * old if old <= MAX_CAP // 2 else MAX_CAP
    return max(required, max(1, doubled))


def _append_values(s, vs, clone=None):
    if s.b:
        return _append_bytes(s, vs)
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
        return s
    if s.b:
        # A same-backing source may overlap the append destination. Preserve
        # its pre-write snapshot; distinct backing can be read through a view.
        source = t.a[t.o:t.o + t.l] if s.a is t.a else memoryview(t.a)[t.o:t.o + t.l]
        return _append_bytes(s, source)
    vs = [(clone(v) if clone else v) for v in t.a[t.o:t.o + t.l]]
    return _append_values(s, vs, clone)


def append_string(b, s):
    return _append_bytes(b, s) if b.b else _append_values(b, list(s))


def _append_bytes(s, vs):
    if not len(vs):
        return s
    n = s.l + len(vs)
    if n <= s.c:
        a, o, c = s.a, s.o, s.c
    else:
        c = grow_cap(s.c, n)
        if c > 2**32 - 1:
            raise fault("slice growth to %d elements exceeds host limits" % c)
        a, o = alloc_bytes(c), 0
        if s.l:
            memoryview(a)[:s.l] = memoryview(s.a)[s.o:s.o + s.l]
    # Byte spread sources are native snapshots, so overlap is safe in either
    # direction. A memoryview assignment cannot resize existing backing.
    memoryview(a)[o + s.l:o + n] = vs
    return Slice(a, o, n, c)

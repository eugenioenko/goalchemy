"""core.slice.store: write s[i] = v into shared backing storage."""

from ..types.panic import idx


def sset(s, i, v):
    s.a[s.o + idx(i, s.l)] = v

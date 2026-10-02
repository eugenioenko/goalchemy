"""core.slice.store: write s[i] = v & 255 if s.b else v into shared backing storage."""

from ..types.panic import idx


def sset(s, i, v):
    s.a[s.o + idx(i, s.l)] = v & 255 if s.b else v

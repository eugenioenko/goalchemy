"""core.slice.index: read s[i], checking the length."""

from ..types.panic import idx


def sget(s, i):
    return s.a[s.o + idx(i, s.l)]

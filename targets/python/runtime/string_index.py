"""core.string.index: byte at an index."""

from ..types.panic import idx


def sindex(s, i):
    return s[idx(i, len(s))]

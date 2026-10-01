"""core.string.from_runes: string(r) for a rune slice."""

from ..types.utf8 import encode


def from_runes(r):
    return b"".join(encode(r.a[r.o + i]) for i in range(r.l))

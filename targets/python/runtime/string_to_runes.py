"""core.string.to_runes: []rune(s) with capacity equal to length."""

from ..types.slice import Slice
from ..types.utf8 import decode


def to_runes(s):
    a = []
    i = 0
    while i < len(s):
        r, n = decode(s, i)
        a.append(r)
        i += n
    return Slice(a, 0, len(a), len(a))

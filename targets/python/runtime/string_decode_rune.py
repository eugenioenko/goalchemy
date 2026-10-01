"""core.string.decode_rune: one rune at a byte offset, as range does."""

from ..types.utf8 import decode


def decode_rune(s, i):
    return decode(s, i)

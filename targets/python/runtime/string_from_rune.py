"""core.string.from_rune: string(x) for an integer, encoding one code point."""

from ..types.utf8 import encode


def from_rune(x):
    return encode(x)

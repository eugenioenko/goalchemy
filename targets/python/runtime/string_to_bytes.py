"""core.string.to_bytes: []byte(s) with capacity equal to length."""

from ..types.slice import Slice


def to_bytes(s):
    a = list(s)
    return Slice(a, 0, len(a), len(a))

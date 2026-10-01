"""Integer kinds: Python ints normalized to their Go width and signedness."""

from .panic import runtime_panic


def wrap8(x):
    return ((x + 0x80) & 0xFF) - 0x80


def wrap16(x):
    return ((x + 0x8000) & 0xFFFF) - 0x8000


def wrap32(x):
    return ((x + 0x80000000) & 0xFFFFFFFF) - 0x80000000


def wrap64(x):
    return ((x + 0x8000000000000000) & 0xFFFFFFFFFFFFFFFF) - 0x8000000000000000


def wrapu8(x):
    return x & 0xFF


def wrapu16(x):
    return x & 0xFFFF


def wrapu32(x):
    return x & 0xFFFFFFFF


def wrapu64(x):
    return x & 0xFFFFFFFFFFFFFFFF


def shift_count(n):
    """Validates a shift count of any integer kind, clamped to 64."""
    if n < 0:
        raise runtime_panic("negative shift amount")
    return 64 if n > 64 else n


def div_zero():
    raise runtime_panic("integer divide by zero")


def tdiv(a, b):
    """Division truncating toward zero."""
    q = abs(a) // abs(b)
    return q if (a < 0) == (b < 0) else -q

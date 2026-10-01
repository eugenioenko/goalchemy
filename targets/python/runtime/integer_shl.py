"""core.integer.shl: left shift by any integer count; negative counts panic."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def shl_i8(a, c):
    n = shift_count(c)
    if n >= 8:
        return 0
    return wrap8(a << n)


def shl_i16(a, c):
    n = shift_count(c)
    if n >= 16:
        return 0
    return wrap16(a << n)


def shl_i32(a, c):
    n = shift_count(c)
    if n >= 32:
        return 0
    return wrap32(a << n)


def shl_i64(a, c):
    n = shift_count(c)
    if n >= 64:
        return 0
    return wrap64(a << n)


def shl_u8(a, c):
    n = shift_count(c)
    if n >= 8:
        return 0
    return wrapu8(a << n)


def shl_u16(a, c):
    n = shift_count(c)
    if n >= 16:
        return 0
    return wrapu16(a << n)


def shl_u32(a, c):
    n = shift_count(c)
    if n >= 32:
        return 0
    return wrapu32(a << n)


def shl_u64(a, c):
    n = shift_count(c)
    if n >= 64:
        return 0
    return wrapu64(a << n)

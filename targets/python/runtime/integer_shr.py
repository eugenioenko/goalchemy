"""core.integer.shr: arithmetic right shift for signed kinds, logical for unsigned."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def shr_i8(a, c):
    n = shift_count(c)
    if n >= 8:
        return -1 if a < 0 else 0
    return a >> n


def shr_i16(a, c):
    n = shift_count(c)
    if n >= 16:
        return -1 if a < 0 else 0
    return a >> n


def shr_i32(a, c):
    n = shift_count(c)
    if n >= 32:
        return -1 if a < 0 else 0
    return a >> n


def shr_i64(a, c):
    n = shift_count(c)
    if n >= 64:
        return -1 if a < 0 else 0
    return a >> n


def shr_u8(a, c):
    n = shift_count(c)
    if n >= 8:
        return 0
    return a >> n


def shr_u16(a, c):
    n = shift_count(c)
    if n >= 16:
        return 0
    return a >> n


def shr_u32(a, c):
    n = shift_count(c)
    if n >= 32:
        return 0
    return a >> n


def shr_u64(a, c):
    n = shift_count(c)
    if n >= 64:
        return 0
    return a >> n

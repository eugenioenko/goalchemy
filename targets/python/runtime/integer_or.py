"""core.integer.or: bitwise OR."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def or_i8(a, b):
    return wrap8(a | b)


def or_i16(a, b):
    return wrap16(a | b)


def or_i32(a, b):
    return wrap32(a | b)


def or_i64(a, b):
    return wrap64(a | b)


def or_u8(a, b):
    return wrapu8(a | b)


def or_u16(a, b):
    return wrapu16(a | b)


def or_u32(a, b):
    return wrapu32(a | b)


def or_u64(a, b):
    return wrapu64(a | b)

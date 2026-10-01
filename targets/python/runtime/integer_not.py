"""core.integer.not: bitwise complement."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def not_i8(a):
    return wrap8(~a)


def not_i16(a):
    return wrap16(~a)


def not_i32(a):
    return wrap32(~a)


def not_i64(a):
    return wrap64(~a)


def not_u8(a):
    return wrapu8(~a)


def not_u16(a):
    return wrapu16(~a)


def not_u32(a):
    return wrapu32(~a)


def not_u64(a):
    return wrapu64(~a)

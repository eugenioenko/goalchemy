"""core.integer.add: wrapping addition."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def add_i8(a, b):
    return wrap8(a + b)


def add_i16(a, b):
    return wrap16(a + b)


def add_i32(a, b):
    return wrap32(a + b)


def add_i64(a, b):
    return wrap64(a + b)


def add_u8(a, b):
    return wrapu8(a + b)


def add_u16(a, b):
    return wrapu16(a + b)


def add_u32(a, b):
    return wrapu32(a + b)


def add_u64(a, b):
    return wrapu64(a + b)

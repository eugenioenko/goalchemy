"""core.integer.rem: remainder with the sign of the dividend."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def rem_i8(a, b):
    if b == 0:
        div_zero()
    return wrap8(a - b * tdiv(a, b))


def rem_i16(a, b):
    if b == 0:
        div_zero()
    return wrap16(a - b * tdiv(a, b))


def rem_i32(a, b):
    if b == 0:
        div_zero()
    return wrap32(a - b * tdiv(a, b))


def rem_i64(a, b):
    if b == 0:
        div_zero()
    return wrap64(a - b * tdiv(a, b))


def rem_u8(a, b):
    if b == 0:
        div_zero()
    return wrapu8(a - b * tdiv(a, b))


def rem_u16(a, b):
    if b == 0:
        div_zero()
    return wrapu16(a - b * tdiv(a, b))


def rem_u32(a, b):
    if b == 0:
        div_zero()
    return wrapu32(a - b * tdiv(a, b))


def rem_u64(a, b):
    if b == 0:
        div_zero()
    return wrapu64(a - b * tdiv(a, b))

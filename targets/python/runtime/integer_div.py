"""core.integer.div: truncating division; a zero divisor panics."""

from ..types.integer import wrap8, wrap16, wrap32, wrap64, wrapu8, wrapu16, wrapu32, wrapu64, shift_count, div_zero, tdiv


def div_i8(a, b):
    if b == 0:
        div_zero()
    return wrap8(tdiv(a, b))


def div_i16(a, b):
    if b == 0:
        div_zero()
    return wrap16(tdiv(a, b))


def div_i32(a, b):
    if b == 0:
        div_zero()
    return wrap32(tdiv(a, b))


def div_i64(a, b):
    if b == 0:
        div_zero()
    return wrap64(tdiv(a, b))


def div_u8(a, b):
    if b == 0:
        div_zero()
    return wrapu8(tdiv(a, b))


def div_u16(a, b):
    if b == 0:
        div_zero()
    return wrapu16(tdiv(a, b))


def div_u32(a, b):
    if b == 0:
        div_zero()
    return wrapu32(tdiv(a, b))


def div_u64(a, b):
    if b == 0:
        div_zero()
    return wrapu64(tdiv(a, b))

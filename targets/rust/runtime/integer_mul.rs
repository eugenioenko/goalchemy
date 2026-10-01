//! core.integer.mul: wrapping multiplication.
use super::*;

pub fn mul_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w8(a.wrapping_mul(b)))
}

pub fn mul_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w16(a.wrapping_mul(b)))
}

pub fn mul_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w32(a.wrapping_mul(b)))
}

pub fn mul_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w64(a.wrapping_mul(b)))
}

pub fn mul_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu8(a.wrapping_mul(b)))
}

pub fn mul_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu16(a.wrapping_mul(b)))
}

pub fn mul_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu32(a.wrapping_mul(b)))
}

pub fn mul_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu64(a.wrapping_mul(b)))
}


//! core.integer.and: bitwise and.
use super::*;

pub fn and_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w8(a & b))
}

pub fn and_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w16(a & b))
}

pub fn and_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w32(a & b))
}

pub fn and_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w64(a & b))
}

pub fn and_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu8(a & b))
}

pub fn and_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu16(a & b))
}

pub fn and_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu32(a & b))
}

pub fn and_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu64(a & b))
}


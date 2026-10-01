//! core.integer.sub: wrapping subtraction.
use super::*;

pub fn sub_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w8(a.wrapping_sub(b)))
}

pub fn sub_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w16(a.wrapping_sub(b)))
}

pub fn sub_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w32(a.wrapping_sub(b)))
}

pub fn sub_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w64(a.wrapping_sub(b)))
}

pub fn sub_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu8(a.wrapping_sub(b)))
}

pub fn sub_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu16(a.wrapping_sub(b)))
}

pub fn sub_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu32(a.wrapping_sub(b)))
}

pub fn sub_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu64(a.wrapping_sub(b)))
}


//! core.integer.or: bitwise or.
use super::*;

pub fn or_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w8(a | b))
}

pub fn or_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w16(a | b))
}

pub fn or_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w32(a | b))
}

pub fn or_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w64(a | b))
}

pub fn or_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu8(a | b))
}

pub fn or_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu16(a | b))
}

pub fn or_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu32(a | b))
}

pub fn or_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu64(a | b))
}


//! core.integer.add: wrapping addition.
use super::*;

pub fn add_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w8(a.wrapping_add(b)))
}

pub fn add_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w16(a.wrapping_add(b)))
}

pub fn add_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w32(a.wrapping_add(b)))
}

pub fn add_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(w64(a.wrapping_add(b)))
}

pub fn add_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu8(a.wrapping_add(b)))
}

pub fn add_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu16(a.wrapping_add(b)))
}

pub fn add_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu32(a.wrapping_add(b)))
}

pub fn add_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    V::Int(wu64(a.wrapping_add(b)))
}


//! core.integer.div: truncated division; division by zero panics.
use super::*;

pub fn div_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w8(a.wrapping_div(b)))
}

pub fn div_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w16(a.wrapping_div(b)))
}

pub fn div_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w32(a.wrapping_div(b)))
}

pub fn div_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w64(a.wrapping_div(b)))
}

pub fn div_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(wu8(a.wrapping_div(b)))
}

pub fn div_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(wu16(a.wrapping_div(b)))
}

pub fn div_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(wu32(a.wrapping_div(b)))
}

pub fn div_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(((a as u64) / (b as u64)) as i64)
}


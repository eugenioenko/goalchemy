//! core.integer.rem: truncated remainder; division by zero panics.
use super::*;

pub fn rem_i8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w8(a.wrapping_rem(b)))
}

pub fn rem_i16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w16(a.wrapping_rem(b)))
}

pub fn rem_i32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w32(a.wrapping_rem(b)))
}

pub fn rem_i64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(w64(a.wrapping_rem(b)))
}

pub fn rem_u8(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(wu8(a.wrapping_rem(b)))
}

pub fn rem_u16(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(wu16(a.wrapping_rem(b)))
}

pub fn rem_u32(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(wu32(a.wrapping_rem(b)))
}

pub fn rem_u64(a: V, b: V) -> V {
    let (a, b) = (a.i(), b.i());
    if b == 0 {
        div_zero();
    }
    V::Int(((a as u64) % (b as u64)) as i64)
}


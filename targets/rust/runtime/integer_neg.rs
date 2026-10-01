//! core.integer.neg: wrapping negation.
use super::*;

pub fn neg_i8(a: V) -> V {
    V::Int(w8(a.i().wrapping_neg()))
}

pub fn neg_i16(a: V) -> V {
    V::Int(w16(a.i().wrapping_neg()))
}

pub fn neg_i32(a: V) -> V {
    V::Int(w32(a.i().wrapping_neg()))
}

pub fn neg_i64(a: V) -> V {
    V::Int(w64(a.i().wrapping_neg()))
}

pub fn neg_u8(a: V) -> V {
    V::Int(wu8(a.i().wrapping_neg()))
}

pub fn neg_u16(a: V) -> V {
    V::Int(wu16(a.i().wrapping_neg()))
}

pub fn neg_u32(a: V) -> V {
    V::Int(wu32(a.i().wrapping_neg()))
}

pub fn neg_u64(a: V) -> V {
    V::Int(wu64(a.i().wrapping_neg()))
}


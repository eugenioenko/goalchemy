//! core.integer.not: bitwise complement.
use super::*;

pub fn not_i8(a: V) -> V {
    V::Int(w8(!a.i()))
}

pub fn not_i16(a: V) -> V {
    V::Int(w16(!a.i()))
}

pub fn not_i32(a: V) -> V {
    V::Int(w32(!a.i()))
}

pub fn not_i64(a: V) -> V {
    V::Int(w64(!a.i()))
}

pub fn not_u8(a: V) -> V {
    V::Int(wu8(!a.i()))
}

pub fn not_u16(a: V) -> V {
    V::Int(wu16(!a.i()))
}

pub fn not_u32(a: V) -> V {
    V::Int(wu32(!a.i()))
}

pub fn not_u64(a: V) -> V {
    V::Int(wu64(!a.i()))
}


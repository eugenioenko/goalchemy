//! core.integer.shl.
use super::*;

/// n is a validated count (count or countu).
pub fn shl_i8(a: V, n: u32) -> V {
    V::Int(if n >= 8 { 0 } else { w8(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_i16(a: V, n: u32) -> V {
    V::Int(if n >= 16 { 0 } else { w16(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_i32(a: V, n: u32) -> V {
    V::Int(if n >= 32 { 0 } else { w32(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_i64(a: V, n: u32) -> V {
    V::Int(if n >= 64 { 0 } else { w64(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_u8(a: V, n: u32) -> V {
    V::Int(if n >= 8 { 0 } else { wu8(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_u16(a: V, n: u32) -> V {
    V::Int(if n >= 16 { 0 } else { wu16(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_u32(a: V, n: u32) -> V {
    V::Int(if n >= 32 { 0 } else { wu32(a.i() << n) })
}

/// n is a validated count (count or countu).
pub fn shl_u64(a: V, n: u32) -> V {
    V::Int(if n >= 64 { 0 } else { wu64(a.i() << n) })
}


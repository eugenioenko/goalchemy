//! core.integer.shr: arithmetic for signed kinds, logical for unsigned.
use super::*;

pub fn shr_i8(a: V, n: u32) -> V {
    let a = a.i();
    V::Int(if n >= 8 { if a < 0 { -1 } else { 0 } } else { a >> n })
}

pub fn shr_i16(a: V, n: u32) -> V {
    let a = a.i();
    V::Int(if n >= 16 { if a < 0 { -1 } else { 0 } } else { a >> n })
}

pub fn shr_i32(a: V, n: u32) -> V {
    let a = a.i();
    V::Int(if n >= 32 { if a < 0 { -1 } else { 0 } } else { a >> n })
}

pub fn shr_i64(a: V, n: u32) -> V {
    let a = a.i();
    V::Int(if n >= 64 { if a < 0 { -1 } else { 0 } } else { a >> n })
}

pub fn shr_u8(a: V, n: u32) -> V {
    V::Int(if n >= 8 { 0 } else { ((a.i() as u64) >> n) as i64 })
}

pub fn shr_u16(a: V, n: u32) -> V {
    V::Int(if n >= 16 { 0 } else { ((a.i() as u64) >> n) as i64 })
}

pub fn shr_u32(a: V, n: u32) -> V {
    V::Int(if n >= 32 { 0 } else { ((a.i() as u64) >> n) as i64 })
}

pub fn shr_u64(a: V, n: u32) -> V {
    V::Int(if n >= 64 { 0 } else { ((a.i() as u64) >> n) as i64 })
}


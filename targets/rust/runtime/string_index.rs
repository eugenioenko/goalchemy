//! core.string.index: s[i] as a byte with bounds checking.
use super::*;

pub fn sindex(x: V, i: V) -> V {
    let b = x.bytes();
    V::Int(b[idx(&i, b.len())] as i64)
}

pub fn sindexu(x: V, i: V) -> V {
    let b = x.bytes();
    V::Int(b[idxu(&i, b.len())] as i64)
}

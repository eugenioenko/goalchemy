//! core.string.to_bytes: []byte(s) with capacity equal to length.
use super::*;

pub fn to_bytes(x: V) -> V {
    let v: Vec<V> = x.bytes().iter().map(|&c| V::Int(c as i64)).collect();
    let n = v.len() as u32;
    V::Slice(alloc(Obj::Vals(v)), 0, n, n)
}

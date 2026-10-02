//! core.string.to_bytes: []byte(s) with capacity equal to length.
use super::*;

pub fn to_bytes(x: V) -> V {
    let v = x.bytes().to_vec();
    let n = v.len() as u32;
    V::ByteSlice(alloc(Obj::Bytes(v)), 0, n, n)
}

//! core.string.to_runes: []rune(s) with capacity equal to length.
use super::*;

pub fn to_runes(x: V) -> V {
    let b = x.bytes();
    let mut v = Vec::new();
    let mut i = 0;
    while i < b.len() {
        let (r, w) = utf8::decode(&b, i);
        v.push(V::Int(r));
        i += w;
    }
    let n = v.len() as u32;
    V::Slice(alloc(Obj::Vals(v)), 0, n, n)
}

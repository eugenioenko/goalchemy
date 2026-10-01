//! core.string.decode_rune: the rune at a byte offset and its width.
use super::*;

pub fn decode_rune(x: V, i: V) -> V {
    let (r, w) = utf8::decode(&x.bytes(), i.i() as usize);
    tuple(vec![V::Int(r), V::Int(w as i64)])
}

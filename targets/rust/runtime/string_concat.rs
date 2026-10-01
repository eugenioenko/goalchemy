//! core.string.concat.
use super::*;

pub fn concat(a: V, b: V) -> V {
    let (a, b) = (a.bytes(), b.bytes());
    if b.is_empty() {
        return V::Str(a);
    }
    if a.is_empty() {
        return V::Str(b);
    }
    let mut v = Vec::with_capacity(a.len() + b.len());
    v.extend_from_slice(&a);
    v.extend_from_slice(&b);
    V::Str(Rc::from(v))
}

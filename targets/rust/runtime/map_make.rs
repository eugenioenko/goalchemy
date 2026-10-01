//! core.map.make.
use super::*;

pub fn make_map(key_of: fn(&V) -> Key) -> V {
    V::Obj(alloc(Obj::Map(GoMap { index: KeyIndex::new(), entries: Vec::new(), key_of })))
}

pub fn identity_key(v: &V) -> Key {
    vkey(v)
}

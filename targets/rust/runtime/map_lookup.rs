//! core.map.lookup: returns (value, ok).
use super::*;

pub fn map_get(m: V, k: V, key_of: fn(&V) -> Key, zero: fn() -> V) -> V {
    let key = key_of(&k);
    if m.is_nil() {
        return tuple(vec![zero(), V::Bool(false)]);
    }
    let found = with_map(&m, |x| x.index.get(&key).map(|&i| x.entries[i].borrow().v.clone()));
    match found {
        Some(v) => tuple(vec![v, V::Bool(true)]),
        None => tuple(vec![zero(), V::Bool(false)]),
    }
}

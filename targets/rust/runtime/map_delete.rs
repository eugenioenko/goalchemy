//! core.map.delete: the key is encoded first, so unhashable keys panic even on nil maps.
use super::*;

pub fn map_delete(m: V, k: V, key_of: fn(&V) -> Key) {
    let key = key_of(&k);
    if m.is_nil() {
        return;
    }
    with_map(&m, |x| {
        if let Some(i) = x.index.remove(&key) {
            x.entries[i].borrow_mut().live = false;
            x.compact();
        }
    })
}

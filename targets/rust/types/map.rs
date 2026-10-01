//! Go maps: a hash index from encoded keys to insertion-ordered entries.
//! Iteration walks a snapshot of the entry list and skips deleted entries.
use super::*;

pub struct Entry {
    pub k: V,
    pub v: V,
    pub live: bool,
}

pub struct GoMap {
    pub index: KeyIndex,
    pub entries: Vec<Rc<RefCell<Entry>>>,
    pub key_of: fn(&V) -> Key,
}

impl GoMap {
    pub fn compact(&mut self) {
        if self.entries.len() > 32 && self.entries.len() > 2 * self.index.len() {
            self.entries.retain(|e| e.borrow().live);
            self.index.clear();
            for (i, e) in self.entries.iter().enumerate() {
                let k = (self.key_of)(&e.borrow().k);
                self.index.insert(k, i);
            }
        }
    }
}

pub struct MapIter {
    pub entries: Vec<Rc<RefCell<Entry>>>,
    pub i: usize,
    pub k: V,
    pub v: V,
}

/// Runs f on a map object.
pub fn with_map<R>(m: &V, f: impl FnOnce(&mut GoMap) -> R) -> R {
    with(m.h(), |o| match o {
        Obj::Map(x) => f(x),
        _ => fault("map expected"),
    })
}

pub fn map_key_of(m: &V) -> fn(&V) -> Key {
    with_map(m, |x| x.key_of)
}

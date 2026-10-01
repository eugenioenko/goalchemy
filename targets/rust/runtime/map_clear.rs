//! core.map.clear.
use super::*;

pub fn map_clear(m: V) {
    if m.is_nil() {
        return;
    }
    with_map(&m, |x| {
        for e in x.entries.iter() {
            e.borrow_mut().live = false;
        }
        x.index.clear();
        x.entries = Vec::new();
    })
}

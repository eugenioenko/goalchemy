//! core.map.store: assignment to a nil map panics.
use super::*;

pub fn map_set(m: V, k: V, v: V) {
    if m.is_nil() {
        plain_panic(b"assignment to entry in nil map");
    }
    let key = map_key_of(&m)(&k);
    with_map(&m, |x| {
        if let Some(&i) = x.index.get(&key) {
            x.entries[i].borrow_mut().v = v;
            return;
        }
        x.entries.push(Rc::new(RefCell::new(Entry { k, v, live: true })));
        x.index.insert(key, x.entries.len() - 1);
    })
}

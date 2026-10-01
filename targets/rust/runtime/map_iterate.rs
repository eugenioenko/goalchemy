//! core.map.iterate: insertion-ordered snapshot iteration that skips deleted entries.
use super::*;

pub fn map_iter(m: V) -> V {
    let entries = if m.is_nil() { Vec::new() } else { with_map(&m, |x| x.entries.clone()) };
    V::Obj(alloc(Obj::Iter(MapIter { entries, i: 0, k: V::Nil, v: V::Nil })))
}

fn with_iter<R>(it: &V, f: impl FnOnce(&mut MapIter) -> R) -> R {
    with(it.h(), |o| match o {
        Obj::Iter(x) => f(x),
        _ => fault("map iterator expected"),
    })
}

pub fn map_next(it: V) -> V {
    V::Bool(with_iter(&it, |x| {
        while x.i < x.entries.len() {
            let e = x.entries[x.i].clone();
            x.i += 1;
            let e = e.borrow();
            if e.live {
                x.k = e.k.clone();
                x.v = e.v.clone();
                return true;
            }
        }
        false
    }))
}

pub fn iter_key(it: V) -> V {
    with_iter(&it, |x| x.k.clone())
}

pub fn iter_val(it: V) -> V {
    with_iter(&it, |x| x.v.clone())
}

pub fn map_keys(m: V) -> V {
    let it = map_iter(m);
    let mut keys = Vec::new();
    while map_next(it.clone()).b() {
        keys.push(iter_key(it.clone()));
    }
    let n = keys.len() as u32;
    V::Slice(alloc(Obj::Vals(keys)), 0, n, n)
}

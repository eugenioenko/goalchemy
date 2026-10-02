//! core.slice.store: s[i] = v with bounds checking.
use super::*;

pub fn sset(x: V, i: V, v: V) {
    let (h, o, l, _, _) = slice_parts(&x);
    let k = idx(&i, l as usize);
    set_slot(h, o as usize + k, v)
}

pub fn ssetu(x: V, i: V, v: V) {
    let (h, o, l, _, _) = slice_parts(&x);
    let k = idxu(&i, l as usize);
    set_slot(h, o as usize + k, v)
}

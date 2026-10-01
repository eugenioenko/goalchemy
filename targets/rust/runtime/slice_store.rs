//! core.slice.store: s[i] = v with bounds checking.
use super::*;

pub fn sset(x: V, i: V, v: V) {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    let k = idx(&i, l as usize);
    set_slot(h, o as usize + k, v)
}

pub fn ssetu(x: V, i: V, v: V) {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    let k = idxu(&i, l as usize);
    set_slot(h, o as usize + k, v)
}

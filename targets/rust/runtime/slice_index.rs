//! core.slice.index: s[i] with bounds checking.
use super::*;

pub fn sget(x: V, i: V) -> V {
    let (h, o, l, _, _) = slice_parts(&x);
    let k = idx(&i, l as usize);
    slot(h, o as usize + k)
}

pub fn sgetu(x: V, i: V) -> V {
    let (h, o, l, _, _) = slice_parts(&x);
    let k = idxu(&i, l as usize);
    slot(h, o as usize + k)
}

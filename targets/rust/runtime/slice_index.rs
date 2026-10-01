//! core.slice.index: s[i] with bounds checking.
use super::*;

pub fn sget(x: V, i: V) -> V {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    let k = idx(&i, l as usize);
    slot(h, o as usize + k)
}

pub fn sgetu(x: V, i: V) -> V {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    let k = idxu(&i, l as usize);
    slot(h, o as usize + k)
}

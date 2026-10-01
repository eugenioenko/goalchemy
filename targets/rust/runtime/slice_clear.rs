//! core.slice.clear: zeroes every element.
use super::*;

pub fn clear_slice(x: V, zero: fn() -> V) {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    for k in 0..l as usize {
        let z = zero();
        set_slot(h, o as usize + k, z);
    }
}

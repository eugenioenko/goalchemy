//! core.slice.clear: zeroes every element.
use super::*;

pub fn clear_slice(x: V, zero: fn() -> V) {
    let (h, o, l, _, bytes) = slice_parts(&x);
    if bytes {
        if l != 0 { with(h, |obj| match obj {
            Obj::Bytes(v) => v[o as usize..(o + l) as usize].fill(0),
            _ => fault("byte backing expected"),
        }); }
        return;
    }
    for k in 0..l as usize {
        let z = zero();
        set_slot(h, o as usize + k, z);
    }
}

//! core.slice.slice: s[lo:hi:max] sharing backing storage; Nil bounds use defaults.
use super::*;

fn bounds(lo: &V, hi: &V, max: &V, len: i64, cap: i64, word: &str, u: bool) -> (i64, i64, i64) {
    let l = if lo.is_nil() { 0 } else { lo.i() };
    let h = if hi.is_nil() { len } else { hi.i() };
    if max.is_nil() {
        check2(l, h, cap, word, u);
        (l, h, cap)
    } else {
        let m = max.i();
        check3(l, h, m, cap, word, u);
        (l, h, m)
    }
}

pub fn reslice(x: V, lo: V, hi: V, max: V, u: bool) -> V {
    let V::Slice(a, o, len, cap) = x else { fault("slice expected") };
    let (l, h, m) = bounds(&lo, &hi, &max, len as i64, cap as i64, "capacity", u);
    if a == 0 {
        return NIL_SLICE;
    }
    V::Slice(a, o + l as u32, (h - l) as u32, (m - l) as u32)
}

/// Slices an array through a pointer: (&a)[lo:hi:max].
pub fn slice_array(arr: V, lo: V, hi: V, max: V, u: bool) -> V {
    let a = nilchk(arr).h();
    let n = vals_len(a) as i64;
    let (l, h, m) = bounds(&lo, &hi, &max, n, n, "length", u);
    V::Slice(a, l as u32, (h - l) as u32, (m - l) as u32)
}

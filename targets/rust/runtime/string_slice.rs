//! core.string.slice: s[lo:hi]; Nil bounds use defaults.
use super::*;

pub fn sslice(x: V, lo: V, hi: V, u: bool) -> V {
    let b = x.bytes();
    let l = if lo.is_nil() { 0 } else { lo.i() };
    let h = if hi.is_nil() { b.len() as i64 } else { hi.i() };
    check2(l, h, b.len() as i64, "length", u);
    s(&b[l as usize..h as usize])
}

//! core.integer.compare: three-way comparison.
use super::*;

pub fn compare_int(a: V, b: V) -> V {
    V::Int(a.i().cmp(&b.i()) as i64)
}

pub fn compare_uint(a: V, b: V) -> V {
    V::Int(cmpu(&a, &b) as i64)
}

//! core.slice.make: make([]T, len, cap) with zeroed capacity.
use super::*;

pub fn make_slice(len: V, cap: V, zero: fn() -> V) -> V {
    let (len, cap) = (len.i(), cap.i());
    if len < 0 || len > (1 << 53) {
        runtime_panic("makeslice: len out of range");
    }
    if cap < len || cap > (1 << 53) {
        runtime_panic("makeslice: cap out of range");
    }
    if cap > u32::MAX as i64 / 2 {
        fault("allocation exceeds host limits");
    }
    let v: Vec<V> = (0..cap).map(|_| zero()).collect();
    V::Slice(alloc(Obj::Vals(v)), 0, len as u32, cap as u32)
}

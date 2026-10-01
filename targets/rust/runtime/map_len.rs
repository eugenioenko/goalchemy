//! core.map.len.
use super::*;

pub fn map_len(m: V) -> V {
    if m.is_nil() {
        return V::Int(0);
    }
    V::Int(with_map(&m, |x| x.index.len()) as i64)
}

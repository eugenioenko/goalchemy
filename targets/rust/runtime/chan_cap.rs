//! core.chan.cap.
use super::*;

pub fn chan_cap(ch: V) -> V {
    if ch.is_nil() {
        return V::Int(0);
    }
    V::Int(with_chan(&ch, |c| c.size) as i64)
}

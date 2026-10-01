//! core.chan.len.
use super::*;

pub fn chan_len(ch: V) -> V {
    if ch.is_nil() {
        return V::Int(0);
    }
    V::Int(with_chan(&ch, |c| c.buf.len()) as i64)
}

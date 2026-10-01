//! std.sync.waitgroup.done.
use super::*;

pub fn std_sync_waitgroup_done(wg: V) {
    std_sync_waitgroup_add(wg, V::Int(-1))
}

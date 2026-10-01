//! std.sync.waitgroup.wait: a pause primitive.
use super::*;

pub fn std_sync_waitgroup_wait(t: &Rc<Task>, wg: V) {
    t.set_rv(Vec::new());
    let wait = with_wg(&wg, |x| {
        if x.n == 0 {
            return false;
        }
        x.waiters.push(t.clone());
        true
    });
    if wait {
        block(t);
    }
}

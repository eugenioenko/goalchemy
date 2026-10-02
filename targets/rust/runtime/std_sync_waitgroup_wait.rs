//! std.sync.waitgroup.wait: a pause primitive.
use super::*;

pub fn std_sync_waitgroup_wait(t: &Rc<Task>, wg: V) {
    check_task(t);
    t.set_rv(Vec::new());
    let wait = with_wg(&wg, |x| {
        if x.n == 0 {
            return false;
        }
        x.waiters.push(t.clone());
        true
    });
    if wait {
        let id = t.id; let owner = t.owner.get();
        let object = wg;
        t.cleanup_roots.replace(vec![object.clone()]);
        t.cleanup.replace(Some(Box::new(move || {
            with_wg(&object, |x| x.waiters.retain(|w| w.id != id || w.owner.get() != owner));
        })));
        block(t);
    }
}

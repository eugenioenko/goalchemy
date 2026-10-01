//! std.sync.mutex.lock: FIFO handoff to waiting tasks.
use super::*;

pub fn with_mutex<R>(m: &V, f: impl FnOnce(&mut Mutex) -> R) -> R {
    with(m.h(), |o| match o {
        Obj::Mutex(x) => f(x),
        _ => fault("mutex expected"),
    })
}

pub fn std_sync_mutex_lock(t: &Rc<Task>, m: V) {
    t.set_rv(Vec::new());
    let wait = with_mutex(&m, |x| {
        if !x.locked {
            x.locked = true;
            return false;
        }
        x.waiters.push(t.clone());
        true
    });
    if wait {
        block(t);
    }
}

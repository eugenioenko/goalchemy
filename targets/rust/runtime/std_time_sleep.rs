//! std.time.sleep: blocks on the virtual clock.
use super::*;

pub fn std_time_sleep(t: &Rc<Task>, d: V) {
    check_task(t);
    t.set_rv(Vec::new());
    let d = d.i();
    if d <= 0 {
        yield_task(t);
        return;
    }
    add_timer(d, Some(t.clone()), None);
    block(t);
}

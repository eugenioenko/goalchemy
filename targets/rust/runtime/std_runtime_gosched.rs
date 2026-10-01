//! std.runtime.gosched: yields to the back of the run queue.
use super::*;

pub fn std_runtime_gosched(t: &Rc<Task>) {
    t.set_rv(Vec::new());
    yield_task(t);
}

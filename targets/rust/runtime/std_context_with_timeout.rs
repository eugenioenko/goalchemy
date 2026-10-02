//! std.context.with_timeout: creation-anchored, earliest inherited deadline.
use super::*;
pub fn std_context_with_timeout(parent: V, d: V) -> V {
    let at = deadline_after(clock_now(), d.i());
    let c = new_child_deadline(parent, Some(at));
    let f = func(-1, cancel_code, vec![c.clone()]);
    tuple(vec![c, f])
}

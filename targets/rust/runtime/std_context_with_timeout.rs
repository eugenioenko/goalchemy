//! std.context.with_timeout: returns (ctx, cancel); the deadline uses the virtual clock.
use super::*;

fn expire(c: V) {
    cancel_ctx(c, context_deadline_exceeded())
}

pub fn std_context_with_timeout(parent: V, d: V) -> V {
    let c = new_child(parent);
    if with_ctx(&c, |x| x.err.is_nil()) {
        add_timer(d.i(), None, Some((expire, c.clone())));
    }
    let f = func(-1, cancel_code, vec![c.clone()]);
    tuple(vec![c, f])
}

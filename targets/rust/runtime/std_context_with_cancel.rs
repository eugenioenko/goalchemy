//! std.context.with_cancel: returns (ctx, cancel).
use super::*;

pub fn cancel_code(env: &[V], _: Vec<V>) -> V {
    cancel_ctx(env[0].clone(), context_canceled());
    V::Nil
}

pub fn std_context_with_cancel(parent: V) -> V {
    let c = new_child(parent);
    let f = func(-1, cancel_code, vec![c.clone()]);
    tuple(vec![c, f])
}

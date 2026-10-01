//! std.context.done.
use super::*;

pub fn std_context_context_done(c: V) -> V {
    with_ctx(&c, |x| x.done.clone())
}

//! std.context.done observes an expired deadline before reading Done.
use super::*;
pub fn std_context_context_done(c: V) -> V { observe_context(&c); with_ctx(&c, |x| x.done.clone()) }

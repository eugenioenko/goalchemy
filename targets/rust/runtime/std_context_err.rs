//! std.context.err and the context tree.
use super::*;

thread_local! {
    static CONTEXT_ERRORS: RefCell<Option<(V, V, V)>> = RefCell::new(None);
}

fn zero_struct() -> V {
    V::Nil
}

/// Canceled, DeadlineExceeded, and Background: allocated once and kept as
/// permanent roots.
fn context_values() -> (V, V, V) {
    if let Some(v) = CONTEXT_ERRORS.with(|c| c.borrow().clone()) {
        return v;
    }
    let canceled = std_errors_new(s(b"context canceled"));
    let deadline = std_errors_new(s(b"context deadline exceeded"));
    let bg = V::Obj(alloc(Obj::Context(Context { done: V::Nil, err: V::Nil, children: Vec::new() })));
    let v = (canceled, deadline, bg);
    for x in [&v.0, &v.1, &v.2] {
        permanent_root(x.clone());
    }
    CONTEXT_ERRORS.with(|c| *c.borrow_mut() = Some(v.clone()));
    v
}

pub fn context_canceled() -> V {
    context_values().0
}

pub fn context_deadline_exceeded() -> V {
    context_values().1
}

pub fn background() -> V {
    context_values().2
}

pub fn with_ctx<R>(c: &V, f: impl FnOnce(&mut Context) -> R) -> R {
    with(c.h(), |o| match o {
        Obj::Context(x) => f(x),
        _ => fault("context expected"),
    })
}

pub fn cancel_ctx(c: V, err: V) {
    let r = with_ctx(&c, |x| {
        if !x.err.is_nil() {
            return None;
        }
        x.err = err.clone();
        Some((x.done.clone(), std::mem::take(&mut x.children)))
    });
    let Some((done, children)) = r else { return };
    chan_close(done);
    for k in children {
        cancel_ctx(k, err.clone());
    }
}

pub fn new_child(parent: V) -> V {
    let done = make_chan(V::Int(0), zero_struct);
    let c = V::Obj(alloc(Obj::Context(Context { done, err: V::Nil, children: Vec::new() })));
    let perr = with_ctx(&parent, |x| x.err.clone());
    if !perr.is_nil() {
        cancel_ctx(c.clone(), perr);
    } else if !veq(&parent, &background()) {
        with_ctx(&parent, |x| x.children.push(c.clone()));
    }
    c
}

pub fn std_context_context_err(c: V) -> V {
    with_ctx(&c, |x| x.err.clone())
}

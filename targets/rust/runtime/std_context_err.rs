//! Source contexts: owner-only state and captured absolute monotonic deadlines.
use super::*;
use std::collections::BTreeMap;
thread_local! { static CONTEXT_ERRORS: RefCell<Option<(V, V, V)>> = RefCell::new(None); }
fn zero_struct() -> V { V::Nil }
fn context_values() -> (V, V, V) {
    if let Some(v) = CONTEXT_ERRORS.with(|c| c.borrow().clone()) { return v }
    let canceled = std_errors_new(s(b"context canceled"));
    let deadline = std_errors_new(s(b"context deadline exceeded"));
    let bg = V::Obj(alloc(Obj::Context(Context {
        owner: 0, thread: std::thread::current().id(), parent: V::Nil, deadline: None, timer: None,
        hooks: BTreeMap::new(), hook_sequence: 0, done: V::Nil, err: V::Nil, children: Vec::new(),
    })));
    let v = (canceled, deadline, bg);
    for x in [&v.0, &v.1, &v.2] { permanent_root(x.clone()); }
    CONTEXT_ERRORS.with(|c| *c.borrow_mut() = Some(v.clone())); v
}
pub fn context_canceled() -> V { context_values().0 }
pub fn context_deadline_exceeded() -> V { context_values().1 }
pub fn background() -> V { context_values().2 }
/// Internal owner retirement may inspect a retired context; source-facing paths
/// first validate the active generation and thread.
pub fn raw_ctx<R>(c: &V, f: impl FnOnce(&mut Context) -> R) -> R {
    with(c.h(), |o| match o { Obj::Context(x) => f(x), _ => fault("context expected") })
}
pub fn check_context(c: &V) {
    let (owner, thread) = raw_ctx(c, |x| (x.owner, x.thread));
    if thread != std::thread::current().id() { host_fault("source context used on native thread") }
    if owner != 0 && !sched(|s| s.active && !s.retiring && s.owner == owner) { host_fault("foreign or retired source context") }
}
pub fn with_ctx<R>(c: &V, f: impl FnOnce(&mut Context) -> R) -> R { check_context(c); raw_ctx(c, f) }
pub fn observe_context(c: &V) {
    check_context(c);
    let now = clock_now();
    let expired = raw_ctx(c, |x| x.err.is_nil() && x.deadline.map_or(false, |d| d <= now));
    if expired { cancel_ctx(c.clone(), context_deadline_exceeded()); }
}
pub fn cancel_ctx(c: V, err: V) {
    check_context(&c);
    let _root = temp_root(&[c.clone(), err.clone()]);
    let r = raw_ctx(&c, |x| {
        if x.owner == 0 || !x.err.is_nil() { return None }
        x.err = err.clone();
        Some((x.done.clone(), std::mem::take(&mut x.children), x.parent.clone(), x.timer.take(), std::mem::take(&mut x.hooks)))
    });
    let Some((done, children, parent, timer, hooks)) = r else { return };
    raw_ctx(&c, |x| x.parent = V::Nil);
    if !parent.is_nil() { raw_ctx(&parent, |x| x.children.retain(|v| !veq(v, &c))); }
    sched(|s| { s.contexts.retain(|v| !veq(v, &c)); if let Some(timer) = timer { s.timers.retain(|t| t.seq != timer); } });
    let mut detached_roots = children.clone(); detached_roots.push(done.clone());
    detached_roots.extend(hooks.values().flat_map(|h| h.roots.iter().cloned()));
    let _detached_roots = temp_root(&detached_roots);
    chan_close(done);
    let mut failure = None;
    for k in children { if let Err(e) = native_catch(|| cancel_ctx(k, err.clone())) { failure.get_or_insert(e); } }
    for (_, hook) in hooks {
        let _roots = temp_root(&hook.roots);
        if let Err(e) = native_catch(hook.call) { failure.get_or_insert(e); }
    }
    if let Some(e) = failure { host_fault(&e); }
}
fn expire_context(c: V) { cancel_ctx(c, context_deadline_exceeded()) }
pub fn new_child(parent: V) -> V { new_child_deadline(parent, None) }
pub fn new_child_deadline(parent: V, requested: Option<i64>) -> V {
    observe_context(&parent);
    let (perr, inherited, root) = with_ctx(&parent, |x| (x.err.clone(), x.deadline, x.owner == 0));
    let deadline = match (requested, inherited) { (Some(a), Some(b)) => Some(a.min(b)), (a, b) => a.or(b) };
    let done = make_chan(V::Int(0), zero_struct);
    let c = V::Obj(alloc(Obj::Context(Context {
        owner: sched(|s| s.owner), thread: std::thread::current().id(),
        parent: if root { V::Nil } else { parent.clone() }, deadline, timer: None,
        hooks: BTreeMap::new(), hook_sequence: 0, done, err: V::Nil, children: Vec::new(),
    })));
    let _roots = temp_root(&[c.clone(), parent.clone(), perr.clone()]);
    sched(|s| s.contexts.push(c.clone()));
    if !root && perr.is_nil() { raw_ctx(&parent, |x| x.children.push(c.clone())); }
    if !perr.is_nil() { cancel_ctx(c.clone(), perr); }
    else if deadline.map_or(false, |d| d <= clock_now()) { cancel_ctx(c.clone(), context_deadline_exceeded()); }
    else if let Some(at) = deadline {
        let timer = add_timer_at(at, None, Some((expire_context, c.clone())));
        raw_ctx(&c, |x| x.timer = Some(timer));
    }
    c
}
pub fn std_context_context_err(c: V) -> V { observe_context(&c); with_ctx(&c, |x| x.err.clone()) }
/// Captured remover validates its creating generation even for Background.
pub struct ContextHookHandle { owner: u64, context: V, sequence: u64 }
impl ContextHookHandle {
    pub fn remove(self) {
        if !sched(|s| s.owner == self.owner && s.active && !s.retiring && s.thread == std::thread::current().id()) { host_fault("foreign or retired context hook remover") }
        if !self.context.is_nil() { with_ctx(&self.context, |x| { x.hooks.remove(&self.sequence); }); }
    }
}
pub fn add_context_hook(c: V, roots: Vec<V>, hook: impl FnOnce() + 'static) -> ContextHookHandle {
    observe_context(&c);
    let owner = sched(|s| s.owner);
    let (background, canceled) = raw_ctx(&c, |x| (x.owner == 0, !x.err.is_nil()));
    if background { return ContextHookHandle { owner, context: V::Nil, sequence: 0 } }
    if canceled { let _root = temp_root(&roots); hook(); return ContextHookHandle { owner, context: V::Nil, sequence: 0 } }
    let sequence = raw_ctx(&c, |x| {
        x.hook_sequence = x.hook_sequence.checked_add(1).unwrap();
        x.hooks.insert(x.hook_sequence, ContextHook { roots, call: Box::new(hook) }); x.hook_sequence
    });
    ContextHookHandle { owner, context: c, sequence }
}

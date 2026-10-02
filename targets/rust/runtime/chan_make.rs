//! core.chan.make: channels with a buffer and FIFO wait queues.
use super::*;
use std::collections::VecDeque;

pub fn recv_done(w: Waiter, v: V, ok: bool) {
    if !w.live() { return }
    let Some(task) = w.task.upgrade() else { return };
    match &w.sel {
        Some(sel) => {
            sel.set(true);
            task.set_rv(vec![V::Int(w.idx as i64), v, V::Bool(ok)]);
        }
        None => task.set_rv(vec![v, V::Bool(ok)]),
    }
    w.clear();
    ready(&task);
}

pub fn send_done(w: Waiter, closed: bool) {
    if !w.live() { return }
    let Some(task) = w.task.upgrade() else { return };
    match &w.sel {
        Some(sel) => {
            sel.set(true);
            task.set_rv(vec![V::Int(w.idx as i64), V::Nil, V::Bool(false)]);
        }
        None => task.set_rv(Vec::new()),
    }
    if closed {
        let p = new_panic(boxv(&PLAIN_ERROR, s(b"send on closed channel")));
        task.resume_panic.replace(p);
    }
    w.clear();
    ready(&task);
}

pub fn dequeue(q: &mut VecDeque<Waiter>) -> Option<Waiter> {
    while let Some(w) = q.pop_front() {
        if w.live() {
            return Some(w);
        }
    }
    None
}

pub fn has_live(q: &VecDeque<Waiter>) -> bool {
    q.iter().any(|w| w.live())
}

/// Receives without blocking: Some((value, ok)) when the receive completed.
pub fn try_recv(ch: &V) -> Option<(V, bool)> {
    enum R {
        Got(V, Option<Waiter>),
        Closed(fn() -> V),
        Wait,
    }
    let r = with_chan(ch, |c| {
        if let Some(v) = c.buf.pop_front() {
            let w = dequeue(&mut c.sendq);
            if let Some(w) = &w {
                c.buf.push_back(w.val.borrow().clone());
            }
            return R::Got(v, w);
        }
        if let Some(w) = dequeue(&mut c.sendq) {
            let value = w.val.borrow().clone();
            return R::Got(value, Some(w));
        }
        if c.closed {
            return R::Closed(c.zero);
        }
        R::Wait
    });
    match r {
        R::Got(v, w) => {
            if let Some(w) = w {
                send_done(w, false);
            }
            Some((v, true))
        }
        R::Closed(z) => Some((z(), false)),
        R::Wait => None,
    }
}

pub fn make_chan(size: V, zero: fn() -> V) -> V {
    let n = size.i();
    if n < 0 || n > (1 << 53) {
        plain_panic(b"makechan: size out of range");
    }
    V::Obj(alloc(Obj::Chan(Chan {
        buf: VecDeque::new(),
        size: n as usize,
        closed: false,
        recvq: VecDeque::new(),
        sendq: VecDeque::new(),
        zero,
    })))
}

/// Registration captures channel handles explicitly for collector traversal.
/// Weak task references avoid a task -> cleanup -> waiter -> task Rc cycle.
pub fn attach_waiter_cleanup(t: &Rc<Task>, registrations: Vec<(V, Waiter)>) {
    check_task(t);
    t.cleanup_roots.replace(registrations.iter().map(|(ch, _)| ch.clone()).collect());
    t.cleanup.replace(Some(Box::new(move || {
        for (ch, w) in registrations {
            with_chan(&ch, |c| {
                c.sendq.retain(|x| !Rc::ptr_eq(x, &w));
                c.recvq.retain(|x| !Rc::ptr_eq(x, &w));
            });
            w.clear();
        }
    })));
}

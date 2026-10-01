//! core.chan.make: channels with a buffer and FIFO wait queues.
use super::*;
use std::collections::VecDeque;

pub fn recv_done(w: Waiter, v: V, ok: bool) {
    match &w.sel {
        Some(sel) => {
            sel.set(true);
            w.task.set_rv(vec![V::Int(w.idx as i64), v, V::Bool(ok)]);
        }
        None => w.task.set_rv(vec![v, V::Bool(ok)]),
    }
    ready(&w.task);
}

pub fn send_done(w: Waiter, closed: bool) {
    match &w.sel {
        Some(sel) => {
            sel.set(true);
            w.task.set_rv(vec![V::Int(w.idx as i64), V::Nil, V::Bool(false)]);
        }
        None => w.task.set_rv(Vec::new()),
    }
    if closed {
        let p = new_panic(boxv(&PLAIN_ERROR, s(b"send on closed channel")));
        w.task.resume_panic.replace(p);
    }
    ready(&w.task);
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
                c.buf.push_back(w.val.clone());
            }
            return R::Got(v, w);
        }
        if let Some(w) = dequeue(&mut c.sendq) {
            return R::Got(w.val.clone(), Some(w));
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

//! core.chan.send: a pause primitive.
use super::*;

pub fn chan_send(t: &Rc<Task>, ch: V, v: V) {
    check_task(t);
    t.set_rv(Vec::new());
    if ch.is_nil() {
        block(t);
        return;
    }
    enum R {
        Closed,
        Recv(Waiter, V),
        Done,
        Wait,
    }
    let mut waiter = None;
    let r = with_chan(&ch, |c| {
        if c.closed {
            return R::Closed;
        }
        if let Some(w) = dequeue(&mut c.recvq) {
            return R::Recv(w, v);
        }
        if c.buf.len() < c.size {
            c.buf.push_back(v);
            return R::Done;
        }
        let w = new_waiter(t, v, None, 0);
        c.sendq.push_back(w.clone()); waiter = Some(w);
        R::Wait
    });
    match r {
        R::Closed => plain_panic(b"send on closed channel"),
        R::Recv(w, v) => recv_done(w, v, true),
        R::Done => {}
        R::Wait => { attach_waiter_cleanup(t, vec![(ch, waiter.unwrap())]); block(t); },
    }
}

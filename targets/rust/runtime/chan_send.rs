//! core.chan.send: a pause primitive.
use super::*;

pub fn chan_send(t: &Rc<Task>, ch: V, v: V) {
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
        c.sendq.push_back(Waiter { task: t.clone(), val: v, sel: None, idx: 0 });
        R::Wait
    });
    match r {
        R::Closed => plain_panic(b"send on closed channel"),
        R::Recv(w, v) => recv_done(w, v, true),
        R::Done => {}
        R::Wait => block(t),
    }
}

//! core.chan.recv: a pause primitive leaving (value, ok).
use super::*;

pub fn chan_recv(t: &Rc<Task>, ch: V) {
    if ch.is_nil() {
        block(t);
        return;
    }
    if let Some((v, ok)) = try_recv(&ch) {
        t.set_rv(vec![v, V::Bool(ok)]);
        return;
    }
    with_chan(&ch, |c| c.recvq.push_back(Waiter { task: t.clone(), val: V::Nil, sel: None, idx: 0 }));
    block(t);
}

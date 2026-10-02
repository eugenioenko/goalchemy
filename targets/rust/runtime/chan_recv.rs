//! core.chan.recv: a pause primitive leaving (value, ok).
use super::*;

pub fn chan_recv(t: &Rc<Task>, ch: V) {
    check_task(t);
    if ch.is_nil() {
        block(t);
        return;
    }
    if let Some((v, ok)) = try_recv(&ch) {
        t.set_rv(vec![v, V::Bool(ok)]);
        return;
    }
    let w = new_waiter(t, V::Nil, None, 0);
    with_chan(&ch, |c| c.recvq.push_back(w.clone()));
    attach_waiter_cleanup(t, vec![(ch, w)]);
    block(t);
}

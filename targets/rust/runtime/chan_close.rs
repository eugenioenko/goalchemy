//! core.chan.close: wakes every receiver and panics blocked senders.
use super::*;

pub fn chan_close(ch: V) {
    if ch.is_nil() {
        plain_panic(b"close of nil channel");
    }
    let r = with_chan(&ch, |c| {
        if c.closed {
            return None;
        }
        c.closed = true;
        let mut rs = Vec::new();
        while let Some(w) = dequeue(&mut c.recvq) {
            rs.push(w);
        }
        let mut ss = Vec::new();
        while let Some(w) = dequeue(&mut c.sendq) {
            ss.push(w);
        }
        Some((rs, ss, c.zero))
    });
    let Some((rs, ss, zero)) = r else { plain_panic(b"close of closed channel") };
    for w in rs {
        recv_done(w, zero(), false);
    }
    for w in ss {
        send_done(w, true);
    }
}

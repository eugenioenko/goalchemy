//! core.select: chooses uniformly among ready cases with the shared choice
//! source; rv is (index, value, ok) with index -1 for default.
use super::*;

/// One case: channel, whether it sends, and the value sent.
pub fn scase(ch: V, send: bool, v: V) -> (V, bool, V) {
    (ch, send, v)
}

pub fn select(t: &Rc<Task>, has_default: bool, cases: Vec<(V, bool, V)>) {
    check_task(t);
    let mut ready_cases = Vec::new();
    for (i, (ch, send, _)) in cases.iter().enumerate() {
        if ch.is_nil() {
            continue;
        }
        let ok = with_chan(ch, |c| {
            if *send {
                c.closed || has_live(&c.recvq) || c.buf.len() < c.size
            } else {
                !c.buf.is_empty() || has_live(&c.sendq) || c.closed
            }
        });
        if ok {
            ready_cases.push(i);
        }
    }
    if !ready_cases.is_empty() {
        let i = ready_cases[choose(ready_cases.len())];
        let (ch, send, v) = cases[i].clone();
        if send {
            let w = with_chan(&ch, |c| {
                if c.closed {
                    return Err(());
                }
                Ok(match dequeue(&mut c.recvq) {
                    Some(w) => Some(w),
                    None => {
                        c.buf.push_back(v.clone());
                        None
                    }
                })
            });
            match w {
                Err(()) => plain_panic(b"send on closed channel"),
                Ok(Some(w)) => recv_done(w, v, true),
                Ok(None) => {}
            }
            t.set_rv(vec![V::Int(i as i64), V::Nil, V::Bool(false)]);
            return;
        }
        let (v, ok) = try_recv(&ch).unwrap_or_else(|| fault("select receive not ready"));
        t.set_rv(vec![V::Int(i as i64), v, V::Bool(ok)]);
        return;
    }
    if has_default {
        t.set_rv(vec![V::Int(-1), V::Nil, V::Bool(false)]);
        return;
    }
    let st = Rc::new(Cell::new(false));
    let mut registrations = Vec::new();
    for (i, (ch, send, v)) in cases.iter().enumerate() {
        if ch.is_nil() {
            continue;
        }
        let w = new_waiter(t, if *send { v.clone() } else { V::Nil }, Some(st.clone()), i);
        with_chan(ch, |c| if *send { c.sendq.push_back(w.clone()) } else { c.recvq.push_back(w.clone()) });
        registrations.push((ch.clone(), w));
    }
    block(t);
    attach_waiter_cleanup(t, registrations);
}

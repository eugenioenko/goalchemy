//! std.sync.waitgroup.add.
use super::*;

pub fn with_wg<R>(w: &V, f: impl FnOnce(&mut WaitGroup) -> R) -> R {
    with(w.h(), |o| match o {
        Obj::WaitGroup(x) => f(x),
        _ => fault("wait group expected"),
    })
}

pub fn std_sync_waitgroup_add(wg: V, d: V) {
    let r = with_wg(&wg, |x| {
        x.n += d.i();
        if x.n < 0 {
            return Err(());
        }
        Ok(if x.n == 0 { std::mem::take(&mut x.waiters) } else { Vec::new() })
    });
    match r {
        Err(()) => throw(boxv(&STRING_TYPE, s(b"sync: negative WaitGroup counter"))),
        Ok(ws) => {
            for t in ws {
                ready(&t);
            }
        }
    }
}

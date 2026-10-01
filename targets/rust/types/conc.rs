//! Channel and synchronization objects.
use super::*;
use std::collections::VecDeque;

pub struct Waiter {
    pub task: Rc<Task>,
    pub val: V,
    pub sel: Option<Rc<Cell<bool>>>,
    pub idx: usize,
}

impl Waiter {
    pub fn live(&self) -> bool {
        self.sel.as_ref().map_or(true, |s| !s.get())
    }
}

pub struct Chan {
    pub buf: VecDeque<V>,
    pub size: usize,
    pub closed: bool,
    pub recvq: VecDeque<Waiter>,
    pub sendq: VecDeque<Waiter>,
    pub zero: fn() -> V,
}

pub struct Mutex {
    pub locked: bool,
    pub waiters: Vec<Rc<Task>>,
}

pub struct WaitGroup {
    pub n: i64,
    pub waiters: Vec<Rc<Task>>,
}

pub struct Context {
    pub done: V,
    pub err: V,
    pub children: Vec<V>,
}

pub fn with_chan<R>(c: &V, f: impl FnOnce(&mut Chan) -> R) -> R {
    with(c.h(), |o| match o {
        Obj::Chan(x) => f(x),
        _ => fault("channel expected"),
    })
}

pub fn new_mutex() -> V {
    V::Obj(alloc(Obj::Mutex(Mutex { locked: false, waiters: Vec::new() })))
}

pub fn new_waitgroup() -> V {
    V::Obj(alloc(Obj::WaitGroup(WaitGroup { n: 0, waiters: Vec::new() })))
}

/// Copies an opaque value object (sync.Mutex, sync.WaitGroup).
pub fn opaque_clone(x: &V) -> V {
    let o = with(x.h(), |o| match o {
        Obj::Mutex(m) => Obj::Mutex(Mutex { locked: m.locked, waiters: m.waiters.clone() }),
        Obj::WaitGroup(w) => Obj::WaitGroup(WaitGroup { n: w.n, waiters: w.waiters.clone() }),
        _ => fault("opaque value expected"),
    });
    V::Obj(alloc(o))
}

pub fn opaque_set(d: &V, src: &V) {
    let o = with(src.h(), |o| match o {
        Obj::Mutex(m) => Obj::Mutex(Mutex { locked: m.locked, waiters: m.waiters.clone() }),
        Obj::WaitGroup(w) => Obj::WaitGroup(WaitGroup { n: w.n, waiters: w.waiters.clone() }),
        _ => fault("opaque value expected"),
    });
    let old = with(d.h(), |x| std::mem::replace(x, o));
    drop(old);
}

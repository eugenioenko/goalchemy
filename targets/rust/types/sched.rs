//! Tasks and resumable frames. A suspending function compiles to a Frame
//! whose locals live in an unrooted Fr traced through its task; step runs
//! it until it returns or reaches a pause point. The scheduler state lives
//! here so the collector can trace every task; the scheduling operations are
//! in core.task.spawn.
use super::*;
use std::collections::VecDeque;

pub type Step = fn(&Rc<Task>, &Rc<Frame>);
pub type Results = fn(&Frame) -> Vec<V>;

pub struct Frame {
    pub l: Fr,
    pub pc: Cell<u32>,
    pub parent: RefCell<Option<Rc<Frame>>>,
    pub panicking: RefCell<V>,
    pub step: Step,
    pub results: Results,
    /// Links used by runtime frames (defer runners).
    pub a: RefCell<Option<Rc<Frame>>>,
    pub b: RefCell<Option<Rc<Frame>>>,
    /// A host primitive run by harness frames.
    pub prim: RefCell<Option<Box<dyn FnOnce(&Rc<Task>)>>>,
}

fn no_results(_: &Frame) -> Vec<V> {
    Vec::new()
}

impl Frame {
    pub fn new(n: usize, step: Step, results: Option<Results>) -> Rc<Frame> {
        Rc::new(Frame {
            l: Fr::unrooted(n),
            pc: Cell::new(0),
            parent: RefCell::new(None),
            panicking: RefCell::new(V::Nil),
            step,
            results: results.unwrap_or(no_results),
            a: RefCell::new(None),
            b: RefCell::new(None),
            prim: RefCell::new(None),
        })
    }
}

pub struct Task {
    pub id: usize,
    pub frame: RefCell<Option<Rc<Frame>>>,
    pub rv: RefCell<Vec<V>>,
    pub blocked: Cell<bool>,
    pub done: Cell<bool>,
    pub resume_panic: RefCell<V>,
    pub cleanup: RefCell<Option<Box<dyn FnOnce()>>>,
    /// Recover state: the panic being handled by running deferred calls
    /// and the identity of the deferred function running.
    pub cur_panic: RefCell<V>,
    pub defer_target: Cell<i64>,
}

impl Task {
    pub fn new(id: usize, frame: Option<Rc<Frame>>) -> Rc<Task> {
        Rc::new(Task {
            id,
            frame: RefCell::new(frame),
            rv: RefCell::new(Vec::new()),
            blocked: Cell::new(false),
            done: Cell::new(false),
            resume_panic: RefCell::new(V::Nil),
            cleanup: RefCell::new(None),
            cur_panic: RefCell::new(V::Nil),
            defer_target: Cell::new(-1),
        })
    }

    pub fn set_rv(&self, v: Vec<V>) {
        let old = std::mem::replace(&mut *self.rv.borrow_mut(), v);
        drop(old);
    }

    pub fn rv(&self, k: usize) -> V {
        self.rv.borrow()[k].clone()
    }
}

pub struct Timer {
    pub at: i64,
    pub seq: i64,
    pub task: Option<Rc<Task>>,
    pub f: Option<(fn(V), V)>,
}

pub struct Sched {
    pub runq: VecDeque<Rc<Task>>,
    pub cur: Rc<Task>,
    pub tasks: Vec<Rc<Task>>,
    pub next_id: usize,
    pub rng: i64,
    pub clock: i64,
    pub timers: Vec<Timer>,
    pub seq: i64,
    pub harness: bool,
}

thread_local! {
    pub static SCHED: RefCell<Sched> = RefCell::new(Sched::new(Task::new(0, None), false));
}

impl Sched {
    pub fn new(main: Rc<Task>, harness: bool) -> Sched {
        Sched {
            runq: VecDeque::new(),
            cur: main.clone(),
            tasks: vec![main],
            next_id: 1,
            rng: seed(),
            clock: 0,
            timers: Vec::new(),
            seq: 0,
            harness,
        }
    }
}

pub fn seed() -> i64 {
    match std::env::var("GOALCHEMY_SEED").ok().and_then(|s| s.parse::<i64>().ok()) {
        Some(v) if v > 0 && v < (1i64 << 32) => v,
        _ => 1,
    }
}

/// Runs f with the scheduler state; f must not call generated code.
pub fn sched<R>(f: impl FnOnce(&mut Sched) -> R) -> R {
    SCHED.with(|s| f(&mut s.borrow_mut()))
}

/// The running task: the holder of recover state.
pub fn cur_task() -> Rc<Task> {
    sched(|s| s.cur.clone())
}

pub fn trace_sched(out: &mut Vec<V>, frames: &mut Vec<Rc<Frame>>) {
    SCHED.with(|s| {
        let s = s.borrow();
        trace_task(&s.cur, out, frames);
        for t in s.tasks.iter().chain(s.runq.iter()) {
            trace_task(t, out, frames);
        }
        for t in s.timers.iter() {
            if let Some(task) = &t.task {
                trace_task(task, out, frames);
            }
            if let Some((_, v)) = &t.f {
                out.push(v.clone());
            }
        }
    })
}

//! core.task.spawn and the cooperative scheduler. A task is a stack of
//! frames driven by a trampoline; exactly one task runs at a time and
//! runnable tasks are dispatched in FIFO order. Pause primitives either
//! complete immediately, leaving their results in the task's rv, or block
//! the task until another task or a timer readies it. Deferred calls,
//! panics, and recover are managed per task.
use super::*;

/// Unwinding payload of a harness case whose task blocked forever.
pub struct BlockedPayload;

/// xorshift32 choice source, identical on every target.
pub fn choose(n: usize) -> usize {
    sched(|s| {
        let mut x = s.rng;
        x ^= (x << 13) & 0xFFFF_FFFF;
        x ^= ((x as u64) >> 17) as i64;
        x ^= (x << 5) & 0xFFFF_FFFF;
        s.rng = x;
        (x % n as i64) as usize
    })
}

pub fn ready(t: &Rc<Task>) {
    sched(|s| s.runq.push_back(t.clone()))
}

pub fn block(t: &Rc<Task>) {
    t.blocked.set(true)
}

pub fn add_timer(d: i64, task: Option<Rc<Task>>, f: Option<(fn(V), V)>) {
    sched(|s| {
        s.seq += 1;
        let (at, seq) = (s.clock + d, s.seq);
        s.timers.push(Timer { at, seq, task, f });
    })
}

pub fn fatal(msg: &str) -> ! {
    out::stderr(format!("fatal error: {}\n", msg).as_bytes());
    std::process::exit(2)
}

fn fire_timers() {
    let due = sched(|s| {
        let at = s.timers.iter().map(|t| t.at).min().unwrap();
        s.clock = at;
        let (mut due, keep): (Vec<Timer>, Vec<Timer>) = s.timers.drain(..).partition(|t| t.at == at);
        s.timers = keep;
        due.sort_by_key(|t| t.seq);
        due
    });
    for t in due {
        if let Some((f, v)) = t.f {
            f(v);
        }
        if let Some(task) = t.task {
            ready(&task);
        }
    }
}

fn next() -> Rc<Task> {
    loop {
        let (t, timers, harness) = sched(|s| (s.runq.pop_front(), !s.timers.is_empty(), s.harness));
        if let Some(t) = t {
            return t;
        }
        if !timers {
            if harness {
                std::panic::resume_unwind(Box::new(BlockedPayload));
            }
            fatal("all goroutines are asleep - deadlock!");
        }
        fire_timers();
    }
}

fn run(t: &Rc<Task>) {
    sched(|s| s.cur = t.clone());
    t.blocked.set(false);
    let c = t.cleanup.borrow_mut().take();
    if let Some(c) = c {
        c();
    }
    while !t.blocked.get() {
        let Some(f) = t.frame.borrow().clone() else { break };
        let rp = t.resume_panic.replace(V::Nil);
        if !rp.is_nil() {
            exit(t, &f, rp);
            continue;
        }
        if let Err(e) = catch(|| (f.step)(t, &f)) {
            exit(t, &f, e);
        }
    }
}

fn set_frame(t: &Task, f: Option<Rc<Frame>>) {
    let old = t.frame.replace(f);
    drop(old);
}

fn exit(t: &Rc<Task>, f: &Rc<Frame>, p: V) {
    if !p.is_nil() {
        let earlier = f.panicking.borrow().clone();
        chain_panic(&p, &earlier);
        f.panicking.replace(p);
    }
    set_frame(t, Some(f.clone()));
    if f.l.has_defers() {
        let r = Frame::new(2, defer_runner_step, None);
        r.a.replace(Some(f.clone()));
        r.parent.replace(Some(f.clone()));
        set_frame(t, Some(r));
        return;
    }
    finish(t, f)
}

fn finish(t: &Rc<Task>, f: &Rc<Frame>) {
    let p = f.panicking.borrow().clone();
    let parent = f.parent.borrow().clone();
    set_frame(t, parent.clone());
    let Some(par) = parent else {
        t.done.set(true);
        sched(|s| s.tasks.retain(|x| !Rc::ptr_eq(x, t)));
        if !p.is_nil() {
            if sched(|s| s.harness) {
                raise(p);
            }
            report_panic(&p);
        }
        return;
    };
    let is_child = par.step as usize == defer_runner_step as Step as usize
        && par.b.borrow().as_ref().map_or(false, |c| Rc::ptr_eq(c, f));
    if is_child {
        child_done(t, &par, p);
        return;
    }
    if !p.is_nil() {
        exit(t, &par, p);
        return;
    }
    t.set_rv((f.results)(f));
}

fn defer_runner_step(t: &Rc<Task>, r: &Rc<Frame>) {
    let tf = r.a.borrow().clone().unwrap();
    while let Some(d) = tf.l.pop_defer() {
        r.l.s(0, t.cur_panic.replace(tf.panicking.borrow().clone()));
        r.l.s(1, V::Int(t.defer_target.replace(d.fid)));
        if d.start {
            let fv = d.f.clone();
            match catch(move || callv(&fv, d.args)) {
                Err(e) => {
                    restore(t, r);
                    after(&tf, e);
                }
                Ok(c) => {
                    let c = c.frame();
                    r.b.replace(Some(c.clone()));
                    c.parent.replace(Some(r.clone()));
                    set_frame(t, Some(c));
                    return;
                }
            }
            continue;
        }
        let fv = d.f.clone();
        let res = catch(move || {
            callv(&fv, d.args);
        });
        restore(t, r);
        after(&tf, res.err().unwrap_or(V::Nil));
    }
    set_frame(t, Some(tf.clone()));
    finish(t, &tf);
}

fn restore(t: &Task, r: &Frame) {
    t.cur_panic.replace(r.l.g(0));
    t.defer_target.set(r.l.g(1).i());
}

fn child_done(t: &Rc<Task>, r: &Rc<Frame>, p: V) {
    restore(t, r);
    r.b.replace(None);
    set_frame(t, Some(r.clone()));
    let tf = r.a.borrow().clone().unwrap();
    after(&tf, p);
}

fn after(tf: &Frame, p: V) {
    if !p.is_nil() {
        let earlier = tf.panicking.borrow().clone();
        chain_panic(&p, &earlier);
        tf.panicking.replace(p);
        return;
    }
    let cur = tf.panicking.borrow().clone();
    if !cur.is_nil() && panic_recovered(&cur) {
        tf.panicking.replace(V::Nil);
    }
}

/// Pushes a callee frame: a pause point.
pub fn call(t: &Rc<Task>, child: V) {
    let c = child.frame();
    c.parent.replace(t.frame.borrow().clone());
    set_frame(t, Some(c));
}

pub fn ret(t: &Rc<Task>, f: &Rc<Frame>) {
    exit(t, f, V::Nil)
}

fn sync_step(t: &Rc<Task>, f: &Rc<Frame>) {
    let args = match f.l.g(1) {
        V::Tuple(a) => a.to_vec(),
        _ => Vec::new(),
    };
    let r = callv(&f.l.g(0), args);
    f.l.s(3, r);
    ret(t, f)
}

fn sync_results(f: &Frame) -> Vec<V> {
    let r = f.l.g(3);
    match f.l.g(2).i() {
        0 => Vec::new(),
        1 => vec![r],
        _ => match r {
            V::Tuple(t) => t.to_vec(),
            _ => fault("results expected"),
        },
    }
}

/// Runs an ordinary call of f with n results as a frame.
pub fn sync_frame(f: V, args: Vec<V>, n: usize) -> V {
    let fr = Frame::new(4, sync_step, Some(sync_results));
    fr.l.s(0, f);
    fr.l.s(1, tuple(args));
    fr.l.s(2, V::Int(n as i64));
    V::Frame(fr)
}

fn adapt_code(env: &[V], args: Vec<V>) -> V {
    sync_frame(env[0].clone(), args, env[1].i() as usize)
}

/// Adapts an ordinary function value with n results to the resumable form.
pub fn adapt(f: V, n: usize) -> V {
    if f.is_nil() {
        return V::Nil;
    }
    func(fid_of(&f), adapt_code, vec![f, V::Int(n as i64)])
}

pub fn adapt_slice(x: V, n: usize) -> V {
    let V::Slice(h, o, l, _) = x else { fault("slice expected") };
    if h == 0 {
        return x;
    }
    let fs: Vec<V> = (0..l as usize).map(|k| slot(h, o as usize + k)).collect();
    let _root = temp_root(&fs);
    let vs: Vec<V> = fs.into_iter().map(|f| adapt(f, n)).collect();
    V::Slice(alloc(Obj::Vals(vs)), 0, l, l)
}

/// go f(args): starts a task running frame f.
pub fn spawn(f: V) {
    let fr = f.frame();
    sched(|s| {
        let t = Task::new(s.next_id, Some(fr));
        s.next_id += 1;
        s.tasks.push(t.clone());
        s.runq.push_back(t);
    })
}

/// go f(args) for an ordinary call: runs it as a frame in a new task.
pub fn spawn_call(f: V, args: Vec<V>) {
    let f = fnchk(f);
    spawn(sync_frame(f, args, 0))
}

fn install(main: Rc<Task>, harness: bool) {
    let old = SCHED.with(|s| std::mem::replace(&mut *s.borrow_mut(), Sched::new(main, harness)));
    drop(old);
}

/// Runs the program entry as the first task until it returns.
pub fn run_main(nglobals: usize, init: fn(), entry: fn() -> V) -> ! {
    run_large(move || {
        init_globals(nglobals);
        init();
        let main = Task::new(0, None);
        install(main.clone(), false);
        let f = match catch(entry) {
            Ok(f) => f,
            Err(p) => report_panic(&p),
        };
        set_frame(&main, Some(f.frame()));
        ready(&main);
        while !main.done.get() {
            let t = next();
            run(&t);
        }
    })
}

/// Requeues the running task: a pause primitive.
pub fn yield_task(t: &Rc<Task>) {
    ready(t);
    block(t);
}

fn await_step(t: &Rc<Task>, f: &Rc<Frame>) {
    if f.pc.get() == 0 {
        f.pc.set(1);
        let p = f.prim.borrow_mut().take().unwrap();
        p(t);
        return;
    }
    let rv = t.rv.borrow().clone();
    f.l.s(0, tuple(rv));
    ret(t, f)
}

fn await_results(f: &Frame) -> Vec<V> {
    match f.l.g(0) {
        V::Tuple(t) => t.to_vec(),
        _ => Vec::new(),
    }
}

/// Runs one pause primitive in an isolated scheduler for a harness case;
/// unwinds with BlockedPayload when no task can run, and with the source
/// panic on panic.
pub fn run_isolated(prim: impl FnOnce(&Rc<Task>) + 'static) -> Vec<V> {
    let h = Frame::new(1, await_step, Some(await_results));
    h.prim.replace(Some(Box::new(prim)));
    let main = Task::new(0, Some(h.clone()));
    install(main.clone(), true);
    ready(&main);
    while !main.done.get() {
        let t = next();
        run(&t);
    }
    await_results(&h)
}

pub fn reset_scheduler() {
    install(Task::new(0, None), true)
}

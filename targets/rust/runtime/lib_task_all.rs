//! lib.task.all: runs each function in its own task and waits for all.
use super::*;

fn all_child_step(t: &Rc<Task>, f: &Rc<Frame>) {
    if f.pc.get() == 0 {
        f.pc.set(1);
        let c = callv(&f.l.g(0), Vec::new());
        call(t, c);
        return;
    }
    std_sync_waitgroup_done(f.l.g(1));
    ret(t, f)
}

pub fn lib_task_all(t: &Rc<Task>, fns: V) {
    t.set_rv(Vec::new());
    let V::Slice(h, o, l, _) = fns else { fault("slice expected") };
    if l == 0 {
        return;
    }
    let wg = new_waitgroup();
    std_sync_waitgroup_add(wg.clone(), V::Int(l as i64));
    for k in 0..l as usize {
        let f = slot(h, o as usize + k);
        let c = Frame::new(2, all_child_step, None);
        c.l.s(0, f);
        c.l.s(1, wg.clone());
        spawn(V::Frame(c));
    }
    with_wg(&wg, |x| x.waiters.push(t.clone()));
    block(t);
}

package rt;

import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.function.Supplier;

/** core.task.spawn and the cooperative scheduler. Suspending functions are
 * compiled to resumable frames: a frame holds the function's locals and the
 * block to resume at, and step runs it until it returns or reaches a pause
 * point. A task is a stack of frames driven by a trampoline; exactly one
 * task runs at a time and runnable tasks are dispatched in FIFO order. Pause
 * primitives either complete immediately, leaving their results in task.rv,
 * or block the task until another task or a timer readies it. Deferred
 * calls, panics, and recover are managed per task by the runtime. */
public final class TaskSpawn {
    private TaskSpawn() {}

    public abstract static class Frame {
        public int pc;
        public ArrayList<Program.Deferred> defers = new ArrayList<>();
        public Frame parent;
        public GoPanic panicking;

        public abstract void step(Task t);

        public Object[] results() {
            return new Object[0];
        }
    }

    public static final class Task extends Program.PanicState {
        final int id;
        public Frame frame;
        public Object[] rv = new Object[0];
        public boolean blocked;
        public boolean done;
        public GoPanic resumePanic;
        public Runnable cleanup;

        Task(int id, Frame frame) {
            this.id = id;
            this.frame = frame;
        }
    }

    /** Thrown out of a harness case when its task blocks with nothing runnable. */
    public static final class Blocked extends RuntimeException {
        Blocked() {
            super("blocked", null, false, false);
        }
    }

    static final class FatalPanic extends RuntimeException {
        final GoPanic p;

        FatalPanic(GoPanic p) {
            super("fatal panic", null, false, false);
            this.p = p;
        }
    }

    record Timer(long at, long seq, Task task, Runnable fn) {}

    public static final class Scheduler {
        final ArrayDeque<Task> runq = new ArrayDeque<>();
        public Task cur;
        final Task main;
        int nextId = 1;
        long rng;
        public long clock;
        final ArrayList<Timer> timers = new ArrayList<>();
        long seq;
        boolean harness;

        Scheduler(Task main) {
            this.cur = main;
            this.main = main;
            this.rng = seed();
        }

        /** xorshift32 choice source, identical on every target. */
        public int choose(int n) {
            long x = rng;
            x ^= (x << 13) & 0xFFFFFFFFL;
            x ^= x >>> 17;
            x ^= (x << 5) & 0xFFFFFFFFL;
            rng = x;
            return (int) (x % n);
        }

        public void ready(Task t) {
            runq.add(t);
        }

        public void block(Task t) {
            t.blocked = true;
        }

        public void addTimer(long d, Task task, Runnable fn) {
            seq++;
            timers.add(new Timer(clock + d, seq, task, fn));
        }

        Task next() {
            while (runq.isEmpty()) {
                if (timers.isEmpty()) {
                    if (harness) throw new Blocked();
                    fatal("all goroutines are asleep - deadlock!");
                }
                fireTimers();
            }
            return runq.poll();
        }

        void fireTimers() {
            long at = Long.MAX_VALUE;
            for (Timer t : timers) at = Math.min(at, t.at());
            clock = at;
            ArrayList<Timer> due = new ArrayList<>();
            ArrayList<Timer> keep = new ArrayList<>();
            for (Timer t : timers) (t.at() == at ? due : keep).add(t);
            timers.clear();
            timers.addAll(keep);
            due.sort((a, b) -> Long.compare(a.seq(), b.seq()));
            for (Timer t : due) {
                if (t.fn() != null) t.fn().run();
                if (t.task() != null) ready(t.task());
            }
        }

        void run(Task t) {
            cur = t;
            t.blocked = false;
            if (t.cleanup != null) {
                Runnable c = t.cleanup;
                t.cleanup = null;
                c.run();
            }
            while (!t.blocked && t.frame != null) {
                GoPanic p = t.resumePanic;
                if (p != null) {
                    t.resumePanic = null;
                    exit(t, t.frame, p);
                    continue;
                }
                Frame f = t.frame;
                try {
                    f.step(t);
                } catch (GoPanic e) {
                    exit(t, f, e);
                }
            }
        }

        void exit(Task t, Frame f, GoPanic p) {
            if (p != null) {
                if (p.prev == null && f.panicking != null && f.panicking != p) p.prev = f.panicking;
                f.panicking = p;
            }
            t.frame = f;
            if (!f.defers.isEmpty()) {
                DeferRunner r = new DeferRunner(f);
                r.parent = f;
                t.frame = r;
                return;
            }
            finish(t, f);
        }

        void finish(Task t, Frame f) {
            GoPanic p = f.panicking;
            Frame parent = f.parent;
            t.frame = parent;
            if (parent == null) {
                t.done = true;
                if (p != null) {
                    if (harness) throw new FatalPanic(p);
                    Program.reportPanic(p);
                }
                return;
            }
            if (parent instanceof DeferRunner r && r.child == f) {
                r.childDone(t, p);
                return;
            }
            if (p != null) {
                exit(t, parent, p);
                return;
            }
            t.rv = f.results();
        }
    }

    static final class DeferRunner extends Frame {
        final Frame target;
        Frame child;
        GoPanic savedPanic;
        Object savedTarget;

        DeferRunner(Frame target) {
            this.target = target;
        }

        public void step(Task t) {
            Frame tf = target;
            while (!tf.defers.isEmpty()) {
                Program.Deferred d = tf.defers.remove(tf.defers.size() - 1);
                savedPanic = t.curPanic;
                savedTarget = t.deferTarget;
                t.curPanic = tf.panicking;
                t.deferTarget = d.fid;
                if (d.start) {
                    Frame c;
                    try {
                        if (d.f == null) throw Panics.nilDeref();
                        c = (Frame) d.f.call(d.args);
                    } catch (GoPanic e) {
                        t.curPanic = savedPanic;
                        t.deferTarget = savedTarget;
                        after(tf, e);
                        continue;
                    }
                    child = c;
                    c.parent = this;
                    t.frame = c;
                    return;
                }
                GoPanic p = null;
                try {
                    if (d.f == null) throw Panics.nilDeref();
                    d.f.call(d.args);
                } catch (GoPanic e) {
                    p = e;
                }
                t.curPanic = savedPanic;
                t.deferTarget = savedTarget;
                after(tf, p);
            }
            t.frame = tf;
            sched.finish(t, tf);
        }

        void childDone(Task t, GoPanic p) {
            t.curPanic = savedPanic;
            t.deferTarget = savedTarget;
            child = null;
            t.frame = this;
            after(target, p);
        }

        void after(Frame tf, GoPanic p) {
            if (p != null) {
                if (p.prev == null && tf.panicking != null && tf.panicking != p) p.prev = tf.panicking;
                tf.panicking = p;
                return;
            }
            if (tf.panicking != null && tf.panicking.recovered) tf.panicking = null;
        }
    }

    static long seed() {
        try {
            String s = System.getenv("GOALCHEMY_SEED");
            long v = s == null ? 1 : Long.parseLong(s);
            return v > 0 && v < (1L << 32) ? v : 1;
        } catch (NumberFormatException e) {
            return 1;
        }
    }

    public static void fatal(String msg) {
        Out.stderr("fatal error: " + msg + "\n");
        System.exit(2);
    }

    public static Scheduler sched = new Scheduler(new Task(0, null));

    public static void call(Task t, Frame child) {
        child.parent = t.frame;
        t.frame = child;
    }

    public static void ret(Task t, Frame f) {
        sched.exit(t, f, null);
    }

    static final class SyncFrame extends Frame {
        final Supplier<Object[]> fn;
        Object[] res = new Object[0];

        SyncFrame(Supplier<Object[]> fn) {
            this.fn = fn;
        }

        public void step(Task t) {
            res = fn.get();
            ret(t, this);
        }

        public Object[] results() {
            return res;
        }
    }

    /** Runs an ordinary call as a frame. */
    public static Frame sync(Supplier<Object[]> fn) {
        return new SyncFrame(fn);
    }

    /** Adapts an ordinary function value with n results to the resumable form. */
    public static Fn adapt(Fn f, int n) {
        if (f == null) return null;
        return Program.closure(f.fid(), a -> sync(() -> {
            Object r = f.call(a);
            return n == 0 ? new Object[0] : n == 1 ? new Object[] {r} : (Object[]) r;
        }));
    }

    public static Slice adaptSlice(Slice s, int n) {
        if (s.a == null) return s;
        Object[] a = new Object[s.l];
        for (int i = 0; i < s.l; i++) a[i] = adapt((Fn) s.a[s.o + i], n);
        return new Slice(a, 0, a.length, a.length);
    }

    /** go f(args): starts a task running frame f. */
    public static void spawn(Frame f) {
        Task t = new Task(sched.nextId++, f);
        sched.ready(t);
    }

    static Scheduler install(Task main, boolean harness) {
        Scheduler s = new Scheduler(main);
        s.harness = harness;
        sched = s;
        Program.panicState = () -> sched.cur;
        return s;
    }

    /** Runs the program entry as the first task until it returns. */
    public static void runMain(Frame entry) {
        Program.runLarge(() -> {
            Task main = new Task(0, entry);
            Scheduler s = install(main, false);
            s.ready(main);
            while (!main.done) s.run(s.next());
        });
        System.out.flush();
        System.exit(0);
    }

    /** Requeues the running task: a pause primitive. */
    public static void yieldTask(Task t) {
        sched.ready(t);
        sched.block(t);
    }

    public interface Primitive {
        void run(Task t);
    }

    static final class AwaitFrame extends Frame {
        final Primitive fn;
        Object[] res = new Object[0];

        AwaitFrame(Primitive fn) {
            this.fn = fn;
        }

        public void step(Task t) {
            if (pc == 0) {
                pc = 1;
                fn.run(t);
                return;
            }
            res = t.rv;
            ret(t, this);
        }

        public Object[] results() {
            return res;
        }
    }

    /** Runs one pause primitive in an isolated scheduler for a harness case;
     * throws Blocked when no task can run, and the source panic on panic. */
    public static Object[] runIsolated(Primitive fn) {
        AwaitFrame h = new AwaitFrame(fn);
        Task main = new Task(0, h);
        Scheduler s = install(main, true);
        s.ready(main);
        try {
            while (!main.done) s.run(s.next());
        } catch (FatalPanic e) {
            throw e.p;
        }
        return h.res;
    }

    public static void resetScheduler() {
        install(new Task(0, null), true);
    }
}

namespace Rt;

/// <summary>A resumable activation of a suspending function: its locals live
/// in subclass fields and pc is the block to resume at.</summary>
public abstract class Frame
{
    public int pc;
    public List<Deferred> defers = new();
    public Frame parent;
    public GoPanic panicking;

    public abstract void step(GoTask t);

    public virtual object[] results() => Array.Empty<object>();
}

public sealed class GoTask : PanicState
{
    internal readonly int id;
    public Frame frame;
    public object[] rv = Array.Empty<object>();
    public bool blocked;
    public bool done;
    public GoPanic resumePanic;
    public Action cleanup;

    internal GoTask(int id, Frame frame)
    {
        this.id = id;
        this.frame = frame;
    }
}

/// <summary>Thrown out of a harness case when its task blocks with nothing runnable.</summary>
public sealed class Blocked : Exception
{
    public Blocked() : base("blocked") { }
}

sealed class FatalPanic : Exception
{
    internal readonly GoPanic p;

    internal FatalPanic(GoPanic p) : base("fatal panic") => this.p = p;
}

public sealed class Scheduler
{
    sealed record Timer(long at, long seq, GoTask task, Action fn);

    readonly Queue<GoTask> runq = new();
    public GoTask cur;
    internal readonly GoTask main;
    internal int nextId = 1;
    long rng;
    public long clock;
    readonly List<Timer> timers = new();
    long seq;
    internal bool harness;

    internal Scheduler(GoTask main)
    {
        cur = main;
        this.main = main;
        rng = R.seed();
    }

    /// <summary>xorshift32 choice source, identical on every target.</summary>
    public int choose(int n)
    {
        long x = rng;
        x ^= (x << 13) & 0xFFFFFFFFL;
        x ^= (long)((ulong)x >> 17);
        x ^= (x << 5) & 0xFFFFFFFFL;
        rng = x;
        return (int)(x % n);
    }

    public void ready(GoTask t) => runq.Enqueue(t);

    public void block(GoTask t) => t.blocked = true;

    public void addTimer(long d, GoTask task, Action fn)
    {
        seq++;
        timers.Add(new Timer(clock + d, seq, task, fn));
    }

    internal GoTask next()
    {
        while (runq.Count == 0)
        {
            if (timers.Count == 0)
            {
                if (harness) throw new Blocked();
                R.fatal("all goroutines are asleep - deadlock!");
            }
            fireTimers();
        }
        return runq.Dequeue();
    }

    void fireTimers()
    {
        long at = long.MaxValue;
        foreach (var t in timers) at = Math.Min(at, t.at);
        clock = at;
        var due = timers.FindAll(t => t.at == at);
        timers.RemoveAll(t => t.at == at);
        due.Sort((a, b) => a.seq.CompareTo(b.seq));
        foreach (var t in due)
        {
            t.fn?.Invoke();
            if (t.task != null) ready(t.task);
        }
    }

    internal void run(GoTask t)
    {
        cur = t;
        t.blocked = false;
        if (t.cleanup != null)
        {
            var c = t.cleanup;
            t.cleanup = null;
            c();
        }
        while (!t.blocked && t.frame != null)
        {
            var p = t.resumePanic;
            if (p != null)
            {
                t.resumePanic = null;
                exit(t, t.frame, p);
                continue;
            }
            var f = t.frame;
            try
            {
                f.step(t);
            }
            catch (GoPanic e)
            {
                exit(t, f, e);
            }
        }
    }

    internal void exit(GoTask t, Frame f, GoPanic p)
    {
        if (p != null)
        {
            if (p.prev == null && f.panicking != null && f.panicking != p) p.prev = f.panicking;
            f.panicking = p;
        }
        t.frame = f;
        if (f.defers.Count > 0)
        {
            t.frame = new DeferRunner(f) { parent = f };
            return;
        }
        finish(t, f);
    }

    internal void finish(GoTask t, Frame f)
    {
        var p = f.panicking;
        var parent = f.parent;
        t.frame = parent;
        if (parent == null)
        {
            t.done = true;
            if (p != null)
            {
                if (harness) throw new FatalPanic(p);
                Program.reportPanic(p);
            }
            return;
        }
        if (parent is DeferRunner r && r.child == f)
        {
            r.childDone(t, p);
            return;
        }
        if (p != null)
        {
            exit(t, parent, p);
            return;
        }
        t.rv = f.results();
    }
}

sealed class DeferRunner : Frame
{
    readonly Frame target;
    internal Frame child;
    GoPanic savedPanic;
    object savedTarget;

    internal DeferRunner(Frame target) => this.target = target;

    public override void step(GoTask t)
    {
        var tf = target;
        while (tf.defers.Count > 0)
        {
            var d = tf.defers[tf.defers.Count - 1];
            tf.defers.RemoveAt(tf.defers.Count - 1);
            savedPanic = t.curPanic;
            savedTarget = t.deferTarget;
            t.curPanic = tf.panicking;
            t.deferTarget = d.fid;
            if (d.start)
            {
                Frame c;
                try
                {
                    if (d.f == null) throw Panics.nilDeref();
                    c = (Frame)d.f.Call(d.args);
                }
                catch (GoPanic e)
                {
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
            try
            {
                if (d.f == null) throw Panics.nilDeref();
                d.f.Call(d.args);
            }
            catch (GoPanic e)
            {
                p = e;
            }
            t.curPanic = savedPanic;
            t.deferTarget = savedTarget;
            after(tf, p);
        }
        t.frame = tf;
        R.sched.finish(t, tf);
    }

    internal void childDone(GoTask t, GoPanic p)
    {
        t.curPanic = savedPanic;
        t.deferTarget = savedTarget;
        child = null;
        t.frame = this;
        after(target, p);
    }

    void after(Frame tf, GoPanic p)
    {
        if (p != null)
        {
            if (p.prev == null && tf.panicking != null && tf.panicking != p) p.prev = tf.panicking;
            tf.panicking = p;
            return;
        }
        if (tf.panicking != null && tf.panicking.recovered) tf.panicking = null;
    }
}

sealed class SyncFrame : Frame
{
    readonly Func<object[]> fn;
    object[] res = Array.Empty<object>();

    internal SyncFrame(Func<object[]> fn) => this.fn = fn;

    public override void step(GoTask t)
    {
        res = fn();
        R.ret(t, this);
    }

    public override object[] results() => res;
}

sealed class AwaitFrame : Frame
{
    readonly Action<GoTask> fn;
    internal object[] res = Array.Empty<object>();

    internal AwaitFrame(Action<GoTask> fn) => this.fn = fn;

    public override void step(GoTask t)
    {
        if (pc == 0)
        {
            pc = 1;
            fn(t);
            return;
        }
        res = t.rv;
        R.ret(t, this);
    }

    public override object[] results() => res;
}

/// <summary>core.task.spawn and the cooperative scheduler. Suspending
/// functions are compiled to resumable frames: a frame holds the function's
/// locals and the block to resume at, and step runs it until it returns or
/// reaches a pause point. A task is a stack of frames driven by a trampoline;
/// exactly one task runs at a time and runnable tasks are dispatched in FIFO
/// order. Pause primitives either complete immediately, leaving their results
/// in task.rv, or block the task until another task or a timer readies it.</summary>
public static partial class R
{
    internal static long seed()
    {
        var s = Environment.GetEnvironmentVariable("GOALCHEMY_SEED");
        if (s == null || !long.TryParse(s, out var v)) return 1;
        return v > 0 && v < (1L << 32) ? v : 1;
    }

    public static void fatal(string msg)
    {
        Out.stderr("fatal error: " + msg + "\n");
        Environment.Exit(2);
    }

    public static Scheduler sched = new Scheduler(new GoTask(0, null));

    public static void call(GoTask t, Frame child)
    {
        child.parent = t.frame;
        t.frame = child;
    }

    public static void ret(GoTask t, Frame f) => sched.exit(t, f, null);

    /// <summary>Runs an ordinary call as a frame.</summary>
    public static Frame sync(Func<object[]> fn) => new SyncFrame(fn);

    /// <summary>Adapts an ordinary function value with n results to the resumable form.</summary>
    public static Fn adapt(Fn f, int n)
    {
        if (f == null) return null;
        return new Fn(a => sync(() =>
        {
            var r = f.F(a);
            return n == 0 ? Array.Empty<object>() : n == 1 ? new[] { r } : (object[])r;
        }), f.Fid);
    }

    public static Slice adaptSlice(Slice s, int n)
    {
        if (s.a == null) return s;
        var a = new object[s.l];
        for (int i = 0; i < s.l; i++) a[i] = adapt((Fn)s.a[s.o + i], n);
        return new Slice(a, 0, a.Length, a.Length);
    }

    /// <summary>go f(args): starts a task running frame f.</summary>
    public static void spawn(Frame f) => sched.ready(new GoTask(sched.nextId++, f));

    static Scheduler install(GoTask main, bool harness)
    {
        var s = new Scheduler(main) { harness = harness };
        sched = s;
        Program.panicState = () => sched.cur;
        return s;
    }

    /// <summary>Runs the program entry as the first task until it returns.</summary>
    public static void runMain(Frame entry)
    {
        Program.runLarge(() =>
        {
            var main = new GoTask(0, entry);
            var s = install(main, false);
            s.ready(main);
            while (!main.done) s.run(s.next());
        });
        Environment.Exit(0);
    }

    /// <summary>Requeues the running task: a pause primitive.</summary>
    public static void yieldTask(GoTask t)
    {
        sched.ready(t);
        sched.block(t);
    }

    /// <summary>Runs one pause primitive in an isolated scheduler for a harness
    /// case; throws Blocked when no task can run, and the source panic on panic.</summary>
    public static object[] runIsolated(Action<GoTask> fn)
    {
        var h = new AwaitFrame(fn);
        var main = new GoTask(0, h);
        var s = install(main, true);
        s.ready(main);
        try
        {
            while (!main.done) s.run(s.next());
        }
        catch (FatalPanic e)
        {
            throw e.p;
        }
        return h.res;
    }

    public static void resetScheduler() => install(new GoTask(0, null), true);
}

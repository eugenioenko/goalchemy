namespace Rt;

sealed class AllChild : Frame
{
    readonly Fn f;
    readonly int[] n;
    readonly GoTask waiter;

    internal AllChild(Fn f, int[] n, GoTask waiter)
    {
        this.f = f;
        this.n = n;
        this.waiter = waiter;
    }

    public override void step(GoTask t)
    {
        if (pc == 0)
        {
            pc = 1;
            if (f == null) throw Panics.nilDeref();
            R.call(t, (Frame)f.Call());
            return;
        }
        if (--n[0] == 0) R.sched.ready(waiter);
        R.ret(t, this);
    }
}

/// <summary>lib.task.all: runs each function in its own task and waits for all.</summary>
public static partial class R
{
    public static void libTaskAll(GoTask t, Slice fns)
    {
        t.rv = Array.Empty<object>();
        if (fns.l == 0) return;
        var n = new[] { fns.l };
        for (int i = 0; i < fns.l; i++) spawn(new AllChild((Fn)fns.a[fns.o + i], n, t));
        sched.block(t);
    }
}

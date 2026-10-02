namespace Rt;

public sealed class WaitGroup
{
    internal long n;
    internal List<GoTask> waiters = new();

    public WaitGroup _clone()
    {
        var w = new WaitGroup();
        w._set(this);
        return w;
    }

    public void _set(WaitGroup o)
    {
        n = o.n;
        waiters = new List<GoTask>(o.waiters);
    }
}

/// <summary>std.sync.waitgroup.add.</summary>
public static partial class R
{
    public static void stdSyncWaitgroupAdd(WaitGroup wg, long d)
    {
        sched.assertDriver();
        wg.n += d;
        if (wg.n < 0) throw new GoPanic(new Box(Program.STRING_TYPE, "sync: negative WaitGroup counter"));
        if (wg.n == 0)
        {
            foreach (var t in wg.waiters) sched.ready(t);
            wg.waiters = new List<GoTask>();
        }
    }
}

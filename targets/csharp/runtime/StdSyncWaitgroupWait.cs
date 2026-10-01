namespace Rt;

/// <summary>std.sync.waitgroup.wait: a pause primitive.</summary>
public static partial class R
{
    public static void stdSyncWaitgroupWait(GoTask t, WaitGroup wg)
    {
        t.rv = Array.Empty<object>();
        if (wg.n == 0) return;
        wg.waiters.Add(t);
        sched.block(t);
    }
}

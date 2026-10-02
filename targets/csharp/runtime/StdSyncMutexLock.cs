namespace Rt;

public sealed class GoMutex
{
    internal bool locked;
    internal List<GoTask> waiters = new();

    public GoMutex _clone()
    {
        var m = new GoMutex();
        m._set(this);
        return m;
    }

    public void _set(GoMutex o)
    {
        locked = o.locked;
        waiters = new List<GoTask>(o.waiters);
    }
}

/// <summary>std.sync.mutex.lock: FIFO handoff to waiting tasks.</summary>
public static partial class R
{
    public static void stdSyncMutexLock(GoTask t, GoMutex m)
    {
        sched.assertTask(t);
        t.rv = Array.Empty<object>();
        if (!m.locked)
        {
            m.locked = true;
            return;
        }
        m.waiters.Add(t);
        t.cleanup = () => m.waiters.Remove(t);
        sched.block(t);
    }
}

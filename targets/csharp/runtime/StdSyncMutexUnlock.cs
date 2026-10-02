namespace Rt;

/// <summary>std.sync.mutex.unlock.</summary>
public static partial class R
{
    public static void stdSyncMutexUnlock(GoMutex m)
    {
        sched.assertDriver();
        if (!m.locked) fatal("sync: unlock of unlocked mutex");
        if (m.waiters.Count > 0)
        {
            var w = m.waiters[0];
            m.waiters.RemoveAt(0);
            sched.ready(w);
            return;
        }
        m.locked = false;
    }
}

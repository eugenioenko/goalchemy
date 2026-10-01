namespace Rt;

/// <summary>std.time.sleep: blocks on the virtual clock.</summary>
public static partial class R
{
    public static void stdTimeSleep(GoTask t, long d)
    {
        t.rv = Array.Empty<object>();
        if (d <= 0)
        {
            yieldTask(t);
            return;
        }
        sched.addTimer(d, t, null);
        sched.block(t);
    }
}

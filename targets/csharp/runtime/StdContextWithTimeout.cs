namespace Rt;

/// <summary>std.context.with_timeout: returns {ctx, cancel}; the child inherits the earliest absolute owner deadline.</summary>
public static partial class R
{
    public static object[] stdContextWithTimeout(GoContext parent, long d)
    {
        var c = newChild(parent);
        long at = deadlineAfter(sched.now(), d);
        c.deadline = c.deadline == null ? at : Math.Min(at, c.deadline.Value);
        observe(c);
        if (c.err == null) c.stopTimer = sched.addTimerAt(c.deadline.Value, null, () => cancel(c, CONTEXT_DEADLINE_EXCEEDED));
        var cancelFn = new Fn(a =>
        {
            cancel(c, CONTEXT_CANCELED);
            return null;
        });
        return new object[] { c, cancelFn };
    }
}

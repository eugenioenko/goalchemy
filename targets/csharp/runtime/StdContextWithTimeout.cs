namespace Rt;

/// <summary>std.context.with_timeout: returns {ctx, cancel}; the deadline uses the virtual clock.</summary>
public static partial class R
{
    public static object[] stdContextWithTimeout(GoContext parent, long d)
    {
        var c = newChild(parent);
        if (c.err == null) sched.addTimer(d, null, () => cancel(c, CONTEXT_DEADLINE_EXCEEDED));
        var cancelFn = new Fn(a =>
        {
            cancel(c, CONTEXT_CANCELED);
            return null;
        });
        return new object[] { c, cancelFn };
    }
}

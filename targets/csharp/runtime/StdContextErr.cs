namespace Rt;

/// Contexts and hooks belong to their creating driver. Background is ownerless.
public sealed class GoContext
{
    internal readonly Chan done;
    internal Box err;
    internal readonly HashSet<GoContext> children = new();
    internal readonly HashSet<Action> hooks = new();
    internal Scheduler owner;
    internal GoContext parent;
    internal long? deadline;
    internal Action release, stopTimer;
    internal GoContext(Chan done) => this.done = done;
}
public static partial class R
{
    public static readonly Box CONTEXT_CANCELED = stdErrorsNew("context canceled");
    public static readonly Box CONTEXT_DEADLINE_EXCEEDED = stdErrorsNew("context deadline exceeded");
    public static readonly GoContext BACKGROUND = new GoContext(null);
    public static void observe(GoContext c)
    {
        if (c == null) throw Panics.nilDeref();
        sched.assertDriver();
        if (c.owner != null && c.owner != sched) throw new HostFault("foreign context owner");
        if (c.err == null && c.deadline != null && c.deadline <= sched.now()) cancel(c, CONTEXT_DEADLINE_EXCEEDED);
    }
    public static void cancel(GoContext c, Box err)
    {
        if (c == null) throw Panics.nilDeref();
        if (c == BACKGROUND) return;
        if (c.owner != sched) throw new HostFault("foreign context owner");
        c.owner.assertDriver(); if (c.err != null) return;
        c.err = err;
        c.release?.Invoke(); c.release = null;
        c.stopTimer?.Invoke(); c.stopTimer = null;
        c.parent?.children.Remove(c); c.parent = null;
        chanClose(c.done);
        Exception failure = null;
        foreach (var child in new List<GoContext>(c.children)) try { cancel(child, err); } catch (Exception e) { failure ??= e; }
        c.children.Clear();
        foreach (var hook in new List<Action>(c.hooks)) try { hook(); } catch (Exception e) { failure ??= e; }
        c.hooks.Clear();
        if (failure != null) throw new HostFault("context cancel hook: " + failure);
    }
    internal static GoContext newChild(GoContext parent)
    {
        observe(parent);
        var owner = sched;
        var c = new GoContext(makeChan(0, () => null)) { owner = owner, deadline = parent.deadline };
        Action dispose = () =>
        {
            c.stopTimer?.Invoke(); c.stopTimer = null;
            c.parent?.children.Remove(c); c.parent = null;
            c.children.Clear(); c.hooks.Clear(); c.release = null;
        };
        owner.disposers.Add(dispose); c.release = () => owner.disposers.Remove(dispose);
        if (parent.err != null) cancel(c, parent.err);
        else if (parent != BACKGROUND) { parent.children.Add(c); c.parent = parent; }
        return c;
    }
    public static Action onCancel(GoContext c, Action hook)
    {
        observe(c); if (c == BACKGROUND) return () => {};
        if (c.err != null) { hook(); return () => {}; }
        c.hooks.Add(hook); return () => { c.owner.assertDriver(); c.hooks.Remove(hook); };
    }
    public static Box stdContextContextErr(GoContext c) { observe(c); return c.err; }
}
/// Generic request-only timeout anchored before input copying/submission.
/// Future capability adapters must validate nil/foreign inputs as declared errors.
public sealed class HostBoundary : IDisposable
{
    public readonly long deadline;
    readonly Scheduler owner;
    readonly GoContext context;
    readonly Action cancelNative;
    Box failure;
    Action stopTimer, detach, dispose;
    bool closed;
    public HostBoundary(GoContext context, long timeout, Action cancelNative)
    {
        R.observe(context); owner = R.sched; this.context = context; this.cancelNative = cancelNative;
        long requested = R.deadlineAfter(owner.now(), timeout);
        deadline = context.deadline == null ? requested : Math.Min(requested, context.deadline.Value);
        dispose = Dispose; owner.disposers.Add(dispose);
        detach = R.onCancel(context, () => expire(context.err));
        if (failure == null && deadline <= owner.now()) expire(R.CONTEXT_DEADLINE_EXCEEDED);
        if (failure == null) stopTimer = owner.addTimerAt(deadline, null, () => expire(R.CONTEXT_DEADLINE_EXCEEDED));
    }
    void expire(Box err) { if (closed || failure != null) return; failure = err; cancelNative(); }
    public Box error()
    {
        owner.assertDriver(); R.observe(context);
        if (failure == null && deadline <= owner.now()) expire(R.CONTEXT_DEADLINE_EXCEEDED);
        return failure;
    }
    public void Dispose()
    {
        owner.assertDriver(); if (closed) return; closed = true;
        stopTimer?.Invoke(); stopTimer = null; detach?.Invoke(); detach = null;
        owner.disposers.Remove(dispose); dispose = null;
    }
}

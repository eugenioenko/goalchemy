namespace Rt;

public sealed class GoContext
{
    internal readonly Chan done;
    internal Box err;
    internal List<GoContext> children = new();

    internal GoContext(Chan done) => this.done = done;
}

/// <summary>std.context.err and the context tree.</summary>
public static partial class R
{
    public static readonly Box CONTEXT_CANCELED = stdErrorsNew("context canceled");
    public static readonly Box CONTEXT_DEADLINE_EXCEEDED = stdErrorsNew("context deadline exceeded");
    public static readonly GoContext BACKGROUND = new GoContext(null);

    internal static void cancel(GoContext c, Box err)
    {
        if (c.err != null) return;
        c.err = err;
        chanClose(c.done);
        foreach (var k in c.children) cancel(k, err);
        c.children = new List<GoContext>();
    }

    internal static GoContext newChild(GoContext parent)
    {
        var c = new GoContext(makeChan(0, () => null));
        if (parent.err != null) cancel(c, parent.err);
        else if (parent != BACKGROUND) parent.children.Add(c);
        return c;
    }

    public static Box stdContextContextErr(GoContext c) => c.err;
}

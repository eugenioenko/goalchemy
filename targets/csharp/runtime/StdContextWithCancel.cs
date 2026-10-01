namespace Rt;

/// <summary>std.context.with_cancel: returns {ctx, cancel}.</summary>
public static partial class R
{
    public static object[] stdContextWithCancel(GoContext parent)
    {
        var c = newChild(parent);
        var cancelFn = new Fn(a =>
        {
            cancel(c, CONTEXT_CANCELED);
            return null;
        });
        return new object[] { c, cancelFn };
    }
}

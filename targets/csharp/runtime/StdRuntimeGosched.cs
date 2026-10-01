namespace Rt;

/// <summary>std.runtime.gosched: yields to the back of the run queue.</summary>
public static partial class R
{
    public static void stdRuntimeGosched(GoTask t)
    {
        t.rv = Array.Empty<object>();
        yieldTask(t);
    }
}

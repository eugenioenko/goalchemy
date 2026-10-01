namespace Rt;

/// <summary>std.errors.is: walks the Unwrap chain comparing and calling Is methods.</summary>
public static partial class R
{
    public static bool stdErrorsIs(Box err, Box target)
    {
        if (err == null || target == null) return err == target;
        bool comparable = target.t.Comparable;
        var cur = err;
        while (cur != null)
        {
            if (comparable && cur.t == target.t && cur.t.Eq(cur.v, target.v)) return true;
            if (cur.t.Methods.TryGetValue("Is", out var isFn) && (bool)isFn.Call(cur.v, target)) return true;
            if (!cur.t.Methods.TryGetValue("Unwrap", out var unwrap)) return false;
            cur = (Box)unwrap.Call(cur.v);
        }
        return false;
    }
}

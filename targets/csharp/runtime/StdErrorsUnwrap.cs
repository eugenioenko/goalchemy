namespace Rt;

/// <summary>std.errors.unwrap.</summary>
public static partial class R
{
    public static Box stdErrorsUnwrap(Box err)
    {
        if (err == null) return null;
        return err.t.Methods.TryGetValue("Unwrap", out var unwrap) ? (Box)unwrap.Call(err.v) : null;
    }
}

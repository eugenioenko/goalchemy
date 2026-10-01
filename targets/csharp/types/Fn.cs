namespace Rt;

public delegate object FnBody(object[] a);

/// <summary>A Go function value. Arguments and results are boxed; a function
/// with several results returns an object[]; one with none returns null.
/// Fid is the identity recover compares against.</summary>
public sealed class Fn
{
    public readonly FnBody F;
    public readonly object Fid;

    public Fn(FnBody f, object fid = null)
    {
        F = f;
        Fid = fid;
    }

    public object Call(params object[] a) => F(a);
}

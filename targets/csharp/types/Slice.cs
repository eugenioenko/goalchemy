namespace Rt;

/// <summary>Slice headers over shared backing arrays. Headers are immutable values.</summary>
public sealed class Slice
{
    public readonly object[] a;
    public readonly int o;
    public readonly int l;
    public readonly int c;

    public Slice(object[] a, int o, int l, int c)
    {
        this.a = a;
        this.o = o;
        this.l = l;
        this.c = c;
    }

    public static readonly Slice NIL = new Slice(null, 0, 0, 0);
}

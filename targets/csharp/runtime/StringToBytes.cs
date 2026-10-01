namespace Rt;

/// <summary>core.string.to_bytes: []byte(s) with capacity equal to length.</summary>
public static partial class R
{
    public static Slice toBytes(string s)
    {
        var a = new object[s.Length];
        for (int i = 0; i < a.Length; i++) a[i] = (long)s[i];
        return new Slice(a, 0, a.Length, a.Length);
    }
}

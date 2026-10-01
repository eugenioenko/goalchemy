namespace Rt;

/// <summary>core.string.slice: s[lo:hi]; null bounds use defaults.</summary>
public static partial class R
{
    public static string sslice(string s, long? lo, long? hi, bool u)
    {
        long l = lo ?? 0;
        long h = hi ?? s.Length;
        Bounds.check2(l, h, s.Length, "length", u);
        return s.Substring((int)l, (int)(h - l));
    }
}

namespace Rt;

/// <summary>core.slice.copy: copy(dst, src) with overlap handled as Go does.</summary>
public static partial class R
{
    public static long copy(Slice dst, Slice src, Func<object, object> clone)
    {
        int n = Math.Min(dst.l, src.l);
        if (n == 0) return 0;
        if (dst.bytes) { Array.Copy(src.a, src.o, dst.a, dst.o, n); return n; }
        var tmp = new object[n];
        Array.Copy(src.a, src.o, tmp, 0, n);
        for (int i = 0; i < n; i++) dst.Set(i, clone == null ? tmp[i] : clone(tmp[i]));
        return n;
    }

    public static long copyString(Slice dst, string s)
    {
        int n = Math.Min(dst.l, s.Length);
        for (int i = 0; i < n; i++) ((byte[])dst.a)[dst.o + i] = unchecked((byte)s[i]);
        return n;
    }
}

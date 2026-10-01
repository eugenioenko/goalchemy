namespace Rt;

/// <summary>core.slice.append: append with Goalchemy's growth rule; aggregate
/// elements are cloned when they move.</summary>
public static partial class R
{
    public static long growCap(long old, long required)
    {
        long doubled = old <= long.MaxValue / 2 ? 2 * old : long.MaxValue;
        return Math.Max(required, Math.Max(1, doubled));
    }

    static Slice appendValues(Slice s, object[] vs, Func<object, object> clone)
    {
        int n = s.l + vs.Length;
        if (vs.Length == 0) return s;
        if (n <= s.c)
        {
            Array.Copy(vs, 0, s.a, s.o + s.l, vs.Length);
            return new Slice(s.a, s.o, n, s.c);
        }
        long c = growCap(s.c, n);
        if (c > int.MaxValue - 64) throw Panics.fault("slice growth to " + c + " elements exceeds host limits");
        var a = new object[(int)c];
        for (int i = 0; i < s.l; i++) a[i] = clone == null ? s.a[s.o + i] : clone(s.a[s.o + i]);
        Array.Copy(vs, 0, a, s.l, vs.Length);
        return new Slice(a, 0, n, (int)c);
    }

    public static Slice append(Slice s, object[] vs, Func<object, object> clone) => appendValues(s, vs, clone);

    public static Slice appendSlice(Slice s, Slice t, Func<object, object> clone)
    {
        if (t.l == 0) return s.a == null && t.a == null ? Slice.NIL : s;
        var vs = new object[t.l];
        for (int i = 0; i < t.l; i++) vs[i] = clone == null ? t.a[t.o + i] : clone(t.a[t.o + i]);
        return appendValues(s, vs, clone);
    }

    public static Slice appendString(Slice b, string s)
    {
        var vs = new object[s.Length];
        for (int i = 0; i < vs.Length; i++) vs[i] = (long)s[i];
        return appendValues(b, vs, null);
    }
}

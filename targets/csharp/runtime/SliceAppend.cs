namespace Rt;

/// <summary>core.slice.append: append with Goalchemy's growth rule.</summary>
public static partial class R
{
    public static long growCap(long old, long required)
    {
        long doubled = old <= long.MaxValue / 2 ? 2 * old : long.MaxValue;
        return Math.Max(required, Math.Max(1, doubled));
    }

    static Slice byteAppendSpace(Slice s, int count)
    {
        if (count == 0) return s;
        long needed = (long)s.l + count;
        if (needed <= s.c) return new Slice(s.a, s.o, (int)needed, s.c, true);
        long cap = growCap(s.c, needed);
        if (cap > int.MaxValue - 64) throw Panics.fault("slice growth to " + cap + " elements exceeds host limits");
        var a = new byte[(int)cap];
        if (s.l != 0) Array.Copy(s.a, s.o, a, 0, s.l);
        return new Slice(a, 0, (int)needed, (int)cap);
    }

    public static Slice appendBytes(Slice s, byte[] vs)
    {
        var r = byteAppendSpace(s, vs.Length);
        if (vs.Length != 0) Array.Copy(vs, 0, r.a, r.o + s.l, vs.Length);
        return r;
    }

    static Slice appendValues(Slice s, object[] vs, Func<object, object> clone)
    {
        if (vs.Length == 0) return s;
        long needed = (long)s.l + vs.Length;
        if (needed <= s.c)
        {
            Array.Copy(vs, 0, s.a, s.o + s.l, vs.Length);
            return new Slice(s.a, s.o, (int)needed, s.c);
        }
        long c = growCap(s.c, needed);
        if (c > int.MaxValue - 64) throw Panics.fault("slice growth to " + c + " elements exceeds host limits");
        var a = new object[(int)c];
        for (int i = 0; i < s.l; i++) a[i] = clone == null ? s.Get(i) : clone(s.Get(i));
        Array.Copy(vs, 0, a, s.l, vs.Length);
        return new Slice(a, 0, (int)needed, (int)c);
    }

    public static Slice append(Slice s, object[] vs, Func<object, object> clone) => appendValues(s, vs, clone);

    public static Slice appendSlice(Slice s, Slice t, Func<object, object> clone)
    {
        if (t.l == 0) return s;
        if (s.bytes)
        {
            var r = byteAppendSpace(s, t.l);
            Array.Copy(t.a, t.o, r.a, r.o + s.l, t.l);
            return r;
        }
        var vs = new object[t.l];
        for (int i = 0; i < t.l; i++) vs[i] = clone == null ? t.Get(i) : clone(t.Get(i));
        return appendValues(s, vs, clone);
    }

    public static Slice appendString(Slice b, string s)
    {
        var r = byteAppendSpace(b, s.Length);
        var a = (byte[])r.a;
        for (int i = 0; i < s.Length; i++) a[r.o + b.l + i] = unchecked((byte)s[i]);
        return r;
    }
}

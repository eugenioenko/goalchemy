namespace Rt;

/// <summary>core.slice.slice: s[lo:hi:max] sharing backing storage; null
/// bounds use defaults.</summary>
public static partial class R
{
    public static Slice reslice(Slice s, long? lo, long? hi, long? max, bool u)
    {
        long l = lo ?? 0;
        long h = hi ?? s.l;
        long m = s.c;
        if (max != null)
        {
            Bounds.check3(l, h, max.Value, s.c, "capacity", u);
            m = max.Value;
        }
        else
        {
            Bounds.check2(l, h, s.c, "capacity", u);
        }
        if (s.a == null) return s;
        return new Slice(s.a, s.o + (int)l, (int)(h - l), (int)(m - l));
    }

    /// <summary>Slices an array through a pointer: (&amp;a)[lo:hi:max].</summary>
    public static Slice sliceArray(Array a, long? lo, long? hi, long? max, bool u)
    {
        long l = lo ?? 0;
        long h = hi ?? a.Length;
        long m = a.Length;
        if (max != null)
        {
            Bounds.check3(l, h, max.Value, a.Length, "length", u);
            m = max.Value;
        }
        else
        {
            Bounds.check2(l, h, a.Length, "length", u);
        }
        return new Slice(a, (int)l, (int)(h - l), (int)(m - l));
    }
}

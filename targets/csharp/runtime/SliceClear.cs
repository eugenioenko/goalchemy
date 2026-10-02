namespace Rt;

/// <summary>core.slice.clear: zeroes every element.</summary>
public static partial class R
{
    public static void clearSlice(Slice s, Func<object> zero)
    {
        if (s.bytes) { if (s.l != 0) Array.Clear(s.a, s.o, s.l); return; }
        for (int i = 0; i < s.l; i++) s.Set(i, zero());
    }
}

namespace Rt;

/// <summary>core.slice.clear: zeroes every element.</summary>
public static partial class R
{
    public static void clearSlice(Slice s, Func<object> zero)
    {
        for (int i = 0; i < s.l; i++) s.a[s.o + i] = zero();
    }
}

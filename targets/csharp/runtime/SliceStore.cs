namespace Rt;

/// <summary>core.slice.store: s[i] = v with bounds checking.</summary>
public static partial class R
{
    public static void sset(Slice s, long i, object v) => s.a[s.o + Panics.idx(i, s.l)] = v;

    public static void ssetu(Slice s, long i, object v) => s.a[s.o + Panics.idxu(i, s.l)] = v;
}

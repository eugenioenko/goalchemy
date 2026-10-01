namespace Rt;

/// <summary>core.slice.index: s[i] with bounds checking.</summary>
public static partial class R
{
    public static object sget(Slice s, long i) => s.a[s.o + Panics.idx(i, s.l)];

    public static object sgetu(Slice s, long i) => s.a[s.o + Panics.idxu(i, s.l)];
}

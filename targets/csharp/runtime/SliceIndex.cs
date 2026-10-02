namespace Rt;

/// <summary>core.slice.index: s[i] with signed or unsigned bounds checking.</summary>
public static partial class R
{
    public static object sget(Slice s, long i) => s.Get(Panics.idx(i, s.l));
    public static object sgetu(Slice s, long i) => s.Get(Panics.idxu(i, s.l));
    public static long bget(Slice s, long i) => ((byte[])s.a)[s.o + Panics.idx(i, s.l)];
    public static long bgetu(Slice s, long i) => ((byte[])s.a)[s.o + Panics.idxu(i, s.l)];
}

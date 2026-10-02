namespace Rt;

/// <summary>core.slice.store: s[i] = v with signed or unsigned bounds checking.</summary>
public static partial class R
{
    public static void sset(Slice s, long i, object v) => s.Set(Panics.idx(i, s.l), v);
    public static void ssetu(Slice s, long i, object v) => s.Set(Panics.idxu(i, s.l), v);
    public static void bset(Slice s, long i, long v) => ((byte[])s.a)[s.o + Panics.idx(i, s.l)] = unchecked((byte)v);
    public static void bsetu(Slice s, long i, long v) => ((byte[])s.a)[s.o + Panics.idxu(i, s.l)] = unchecked((byte)v);
}

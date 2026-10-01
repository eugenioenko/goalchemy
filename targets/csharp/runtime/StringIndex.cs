namespace Rt;

/// <summary>core.string.index: s[i] as a byte with bounds checking.</summary>
public static partial class R
{
    public static long sindex(string s, long i) => s[Panics.idx(i, s.Length)];

    public static long sindexu(string s, long i) => s[Panics.idxu(i, s.Length)];
}

namespace Rt;

/// <summary>core.integer.compare: three-way comparison.</summary>
public static partial class R
{
    public static long compare_int(long a, long b) => a.CompareTo(b);

    public static long compare_uint(long a, long b) => ((ulong)a).CompareTo((ulong)b);
}

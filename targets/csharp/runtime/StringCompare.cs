namespace Rt;

/// <summary>core.string.compare: bytewise three-way comparison.</summary>
public static partial class R
{
    public static long scompare(string a, string b) => Math.Sign(string.CompareOrdinal(a, b));
}

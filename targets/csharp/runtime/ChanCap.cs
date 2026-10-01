namespace Rt;

/// <summary>core.chan.cap.</summary>
public static partial class R
{
    public static long chanCap(Chan ch) => ch == null ? 0 : ch.size;
}

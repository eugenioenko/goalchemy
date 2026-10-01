namespace Rt;

/// <summary>core.chan.len.</summary>
public static partial class R
{
    public static long chanLen(Chan ch) => ch == null ? 0 : ch.buf.Count;
}

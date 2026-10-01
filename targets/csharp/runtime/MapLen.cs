namespace Rt;

/// <summary>core.map.len.</summary>
public static partial class R
{
    public static long mapLen(GoMap m) => m == null ? 0 : m.index.Count;
}

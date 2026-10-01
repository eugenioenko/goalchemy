namespace Rt;

/// <summary>core.map.clear.</summary>
public static partial class R
{
    public static void mapClear(GoMap m)
    {
        if (m == null) return;
        foreach (var e in m.entries) e.live = false;
        m.index.Clear();
        m.entries = new List<GoMap.Entry>();
    }
}

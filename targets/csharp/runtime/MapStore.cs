namespace Rt;

/// <summary>core.map.store: assignment to a nil map panics.</summary>
public static partial class R
{
    public static void mapSet(GoMap m, object k, object v)
    {
        if (m == null) throw Panics.plainPanic("assignment to entry in nil map");
        var key = m.keyOf(k);
        if (m.index.TryGetValue(key, out var e))
        {
            e.v = v;
            return;
        }
        m.add(key, k, v);
    }
}

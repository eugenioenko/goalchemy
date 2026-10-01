namespace Rt;

/// <summary>core.map.delete: the key is encoded first, so unhashable keys panic even on nil maps.</summary>
public static partial class R
{
    public static void mapDelete(GoMap m, object k, Func<object, object> keyOf)
    {
        var key = keyOf(k);
        if (m == null) return;
        if (m.index.Remove(key, out var e))
        {
            e.live = false;
            m.compact();
        }
    }
}

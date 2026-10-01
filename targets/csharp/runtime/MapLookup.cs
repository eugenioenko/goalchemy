namespace Rt;

/// <summary>core.map.lookup: returns {value, ok}.</summary>
public static partial class R
{
    public static object[] mapGet(GoMap m, object k, Func<object, object> keyOf, Func<object> zero)
    {
        var key = keyOf(k);
        if (m == null) return new object[] { zero(), false };
        return m.index.TryGetValue(key, out var e) ? new object[] { e.v, true } : new object[] { zero(), false };
    }
}

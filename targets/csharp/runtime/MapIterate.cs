namespace Rt;

/// <summary>core.map.iterate: insertion-ordered snapshot iteration that skips deleted entries.</summary>
public static partial class R
{
    public static GoMap.Iter mapIter(GoMap m) => new GoMap.Iter(m == null ? Array.Empty<GoMap.Entry>() : m.entries.ToArray());

    public static bool mapNext(GoMap.Iter it)
    {
        while (it.i < it.entries.Length)
        {
            var e = it.entries[it.i++];
            if (e.live)
            {
                it.k = e.k;
                it.v = e.v;
                return true;
            }
        }
        return false;
    }

    public static Slice mapKeys(GoMap m)
    {
        var it = mapIter(m);
        var keys = new List<object>();
        while (mapNext(it)) keys.Add(it.k);
        var a = keys.ToArray();
        return new Slice(a, 0, a.Length, a.Length);
    }
}

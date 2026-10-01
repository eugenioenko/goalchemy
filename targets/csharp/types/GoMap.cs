namespace Rt;

/// <summary>Go maps: insertion-ordered entries with snapshot iteration.</summary>
public sealed class GoMap
{
    public sealed class Entry
    {
        public readonly object k;
        public object v;
        public bool live = true;

        public Entry(object k, object v)
        {
            this.k = k;
            this.v = v;
        }
    }

    public readonly Dictionary<object, Entry> index = new();
    public List<Entry> entries = new();
    public readonly Func<object, object> keyOf;

    public GoMap(Func<object, object> keyOf) => this.keyOf = keyOf;

    public void add(object key, object k, object v)
    {
        var e = new Entry(k, v);
        index[key] = e;
        entries.Add(e);
    }

    public void compact()
    {
        if (entries.Count > 32 && entries.Count > 2 * index.Count) entries = entries.FindAll(e => e.live);
    }

    public static object identityKey(object k) => k;

    public sealed class Iter
    {
        public readonly Entry[] entries;
        public int i;
        public object k;
        public object v;

        public Iter(Entry[] entries) => this.entries = entries;
    }
}

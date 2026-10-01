package rt;

/** core.map.iterate: snapshot iteration in insertion order. */
public final class MapIterate {
    private MapIterate() {}

    public static GoMap.Iter mapIter(GoMap m) {
        return new GoMap.Iter(m == null ? new GoMap.Entry[0] : m.entries.toArray(new GoMap.Entry[0]));
    }

    public static boolean mapNext(GoMap.Iter it) {
        while (it.i < it.entries.length) {
            GoMap.Entry e = it.entries[it.i++];
            if (e.live) {
                it.k = e.k;
                it.v = e.v;
                return true;
            }
        }
        return false;
    }

    public static Slice mapKeys(GoMap m) {
        GoMap.Iter it = mapIter(m);
        java.util.ArrayList<Object> keys = new java.util.ArrayList<>();
        while (mapNext(it)) keys.add(it.k);
        Object[] a = keys.toArray();
        return new Slice(a, 0, a.length, a.length);
    }

}

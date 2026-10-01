package rt;

/** core.map.store: m[k] = v; storing into a nil map panics. */
public final class MapStore {
    private MapStore() {}

    public static void mapSet(GoMap m, Object k, Object v) {
        if (m == null) throw Panics.plainPanic("assignment to entry in nil map");
        Object key = m.keyOf.key(k);
        GoMap.Entry e = m.index.get(key);
        if (e != null) {
            e.v = v;
            return;
        }
        m.add(key, k, v);
    }

}

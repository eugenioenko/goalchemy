package rt;

/** core.map.delete: delete(m, k); a nil map is a no-op after hashing. */
public final class MapDelete {
    private MapDelete() {}

    public static void mapDelete(GoMap m, Object k, GoMap.KeyOf keyOf) {
        Object key = keyOf.key(k);
        if (m == null) return;
        GoMap.Entry e = m.index.remove(key);
        if (e != null) {
            e.live = false;
            m.compact();
        }
    }

}

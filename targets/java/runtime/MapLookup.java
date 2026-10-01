package rt;

import java.util.function.Supplier;

/** core.map.lookup: v, ok := m[k]. Nil maps still hash the key. */
public final class MapLookup {
    private MapLookup() {}

    /** Returns {value, ok}. */
    public static Object[] mapGet(GoMap m, Object k, GoMap.KeyOf keyOf, Supplier<Object> zero) {
        Object key = keyOf.key(k);
        if (m == null) return new Object[] {zero.get(), false};
        GoMap.Entry e = m.index.get(key);
        return e == null ? new Object[] {zero.get(), false} : new Object[] {e.v, true};
    }

}

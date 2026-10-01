package rt;

/** core.map.clear: remove every entry. */
public final class MapClear {
    private MapClear() {}

    public static void mapClear(GoMap m) {
        if (m == null) return;
        for (GoMap.Entry e : m.entries) e.live = false;
        m.index.clear();
        m.entries = new java.util.ArrayList<>();
    }

}

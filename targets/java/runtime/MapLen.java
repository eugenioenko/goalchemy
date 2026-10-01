package rt;

/** core.map.len: number of entries. */
public final class MapLen {
    private MapLen() {}

    public static long mapLen(GoMap m) {
        return m == null ? 0 : m.index.size();
    }

}

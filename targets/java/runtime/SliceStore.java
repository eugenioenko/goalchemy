package rt;

/** core.slice.store: write s[i] = v into shared backing storage. */
public final class SliceStore {
    private SliceStore() {}

    public static void sset(Slice s, long i, Object v) {
        s.set(Panics.idx(i, s.l), v);
    }

    public static void ssetu(Slice s, long i, Object v) {
        s.set(Panics.idxu(i, s.l), v);
    }

    public static void bset(Slice s, long i, long v) {
        ((byte[]) s.a)[s.o + Panics.idx(i, s.l)] = (byte) v;
    }

    public static void bsetu(Slice s, long i, long v) {
        ((byte[]) s.a)[s.o + Panics.idxu(i, s.l)] = (byte) v;
    }
}

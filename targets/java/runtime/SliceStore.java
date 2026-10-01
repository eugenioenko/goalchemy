package rt;

/** core.slice.store: write s[i] = v into shared backing storage. */
public final class SliceStore {
    private SliceStore() {}

    public static void sset(Slice s, long i, Object v) {
        s.a[s.o + Panics.idx(i, s.l)] = v;
    }

    public static void ssetu(Slice s, long i, Object v) {
        s.a[s.o + Panics.idxu(i, s.l)] = v;
    }

}

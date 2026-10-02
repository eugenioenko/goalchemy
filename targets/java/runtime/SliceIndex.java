package rt;

/** core.slice.index: read s[i], checking the length. */
public final class SliceIndex {
    private SliceIndex() {}

    public static Object sget(Slice s, long i) {
        return s.get(Panics.idx(i, s.l));
    }

    public static Object sgetu(Slice s, long i) {
        return s.get(Panics.idxu(i, s.l));
    }

    public static long bget(Slice s, long i) {
        return ((byte[]) s.a)[s.o + Panics.idx(i, s.l)] & 255L;
    }

    public static long bgetu(Slice s, long i) {
        return ((byte[]) s.a)[s.o + Panics.idxu(i, s.l)] & 255L;
    }
}

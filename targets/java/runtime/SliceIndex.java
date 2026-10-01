package rt;

/** core.slice.index: read s[i], checking the length. */
public final class SliceIndex {
    private SliceIndex() {}

    public static Object sget(Slice s, long i) {
        return s.a[s.o + Panics.idx(i, s.l)];
    }

    public static Object sgetu(Slice s, long i) {
        return s.a[s.o + Panics.idxu(i, s.l)];
    }

}

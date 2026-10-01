package rt;

/** core.string.index: byte at an index. */
public final class StringIndex {
    private StringIndex() {}

    public static long sindex(String s, long i) {
        return s.charAt(Panics.idx(i, s.length()));
    }

    public static long sindexu(String s, long i) {
        return s.charAt(Panics.idxu(i, s.length()));
    }

}

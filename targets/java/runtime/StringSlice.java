package rt;

/** core.string.slice: substring by byte bounds; null bounds use defaults. */
public final class StringSlice {
    private StringSlice() {}

    public static String sslice(String s, Long lo, Long hi, boolean u) {
        long l = lo == null ? 0 : lo;
        long h = hi == null ? s.length() : hi;
        Bounds.check2(l, h, s.length(), "length", u);
        return s.substring((int) l, (int) h);
    }

}

package rt;

/** core.string.decode_rune: one rune at a byte offset, as range does. */
public final class StringDecodeRune {
    private StringDecodeRune() {}

    /** Returns {rune, width}. */
    public static Object[] decodeRune(String s, long i) {
        int[] r = Utf8.decode(s, (int) i);
        return new Object[] {(long) r[0], (long) r[1]};
    }

}

package rt;

/** core.string.from_rune: string(x) for an integer, encoding one code point. */
public final class StringFromRune {
    private StringFromRune() {}

    public static String fromRune(long x) {
        return Utf8.encode(x);
    }

    /** For unsigned 64-bit operands: negative longs are huge values. */
    public static String fromRuneU(long x) {
        return Utf8.encode(x < 0 ? -1 : x);
    }

}

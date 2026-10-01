package rt;

/** Go slice-bounds checks and messages. When u is set the bounds have an
 * unsigned 64-bit type: negative longs are huge values. */
public final class Bounds {
    private Bounds() {}

    private static RuntimeException oob(String m) {
        return Panics.runtimePanic("slice bounds out of range " + m);
    }

    private static String s(long x, boolean u) {
        return u ? Long.toUnsignedString(x) : Long.toString(x);
    }

    private static boolean neg(long x, boolean u) {
        return !u && x < 0;
    }

    private static boolean gt(long x, long y, boolean u) {
        return u ? Long.compareUnsigned(x, y) > 0 : x > y;
    }

    public static void check2(long lo, long hi, long limit, String word, boolean u) {
        if (neg(hi, u)) throw oob("[:" + s(hi, u) + "]");
        if (gt(hi, limit, u)) throw oob("[:" + s(hi, u) + "] with " + word + " " + limit);
        if (neg(lo, u)) throw oob("[" + s(lo, u) + ":]");
        if (gt(lo, hi, u)) throw oob("[" + s(lo, u) + ":" + s(hi, u) + "]");
    }

    public static void check3(long lo, long hi, long max, long limit, String word, boolean u) {
        if (neg(max, u)) throw oob("[::" + s(max, u) + "]");
        if (gt(max, limit, u)) throw oob("[::" + s(max, u) + "] with " + word + " " + limit);
        if (neg(hi, u)) throw oob("[:" + s(hi, u) + ":]");
        if (gt(hi, max, u)) throw oob("[:" + s(hi, u) + ":" + s(max, u) + "]");
        if (neg(lo, u)) throw oob("[" + s(lo, u) + "::]");
        if (gt(lo, hi, u)) throw oob("[" + s(lo, u) + ":" + s(hi, u) + ":]");
    }
}

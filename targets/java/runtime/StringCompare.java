package rt;

/** core.string.compare: bytewise comparison. */
public final class StringCompare {
    private StringCompare() {}

    public static long scompare(String a, String b) {
        return Integer.signum(a.compareTo(b));
    }

}

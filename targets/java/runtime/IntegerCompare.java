package rt;

/** core.integer.compare: three-way comparison. */
public final class IntegerCompare {
    private IntegerCompare() {}

    public static long compare_int(long a, long b) {
        return Long.compare(a, b);
    }

    public static long compare_uint(long a, long b) {
        return Long.compareUnsigned(a, b);
    }

}

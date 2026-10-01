package rt;

/** Integer kinds: every kind is a long normalized to its Go width; 64-bit
 * unsigned values are stored as their two's-complement bits. */
public final class Ints {
    private Ints() {}

    public static long w8(long x) { return (byte) x; }
    public static long w16(long x) { return (short) x; }
    public static long w32(long x) { return (int) x; }
    public static long w64(long x) { return x; }
    public static long wu8(long x) { return x & 0xFFL; }
    public static long wu16(long x) { return x & 0xFFFFL; }
    public static long wu32(long x) { return x & 0xFFFFFFFFL; }
    public static long wu64(long x) { return x; }

    /** Validates a signed shift count, clamped to 64. */
    public static int count(long n) {
        if (n < 0) throw Panics.runtimePanic("negative shift amount");
        return n > 64 ? 64 : (int) n;
    }

    /** Validates an unsigned shift count, clamped to 64. */
    public static int countu(long n) {
        return n < 0 || n > 64 ? 64 : (int) n;
    }

    public static RuntimeException divZero() {
        return Panics.runtimePanic("integer divide by zero");
    }

    public static String u64String(long x) {
        return Long.toUnsignedString(x);
    }
}

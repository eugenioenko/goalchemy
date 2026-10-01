package rt;

/** core.integer.shr: arithmetic right shift for signed kinds, logical for unsigned. */
public final class IntegerShr {
    private IntegerShr() {}

    public static long shr_i8(long a, int n) {
        if (n >= 8) return a < 0 ? -1 : 0;
        return a >> n;
    }

    public static long shr_i16(long a, int n) {
        if (n >= 16) return a < 0 ? -1 : 0;
        return a >> n;
    }

    public static long shr_i32(long a, int n) {
        if (n >= 32) return a < 0 ? -1 : 0;
        return a >> n;
    }

    public static long shr_i64(long a, int n) {
        if (n >= 64) return a < 0 ? -1 : 0;
        return a >> n;
    }

    public static long shr_u8(long a, int n) {
        if (n >= 8) return 0;
        return a >>> n;
    }

    public static long shr_u16(long a, int n) {
        if (n >= 16) return 0;
        return a >>> n;
    }

    public static long shr_u32(long a, int n) {
        if (n >= 32) return 0;
        return a >>> n;
    }

    public static long shr_u64(long a, int n) {
        if (n >= 64) return 0;
        return a >>> n;
    }

}

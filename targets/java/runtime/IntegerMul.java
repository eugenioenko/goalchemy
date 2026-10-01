package rt;

/** core.integer.mul: wrapping multiplication. */
public final class IntegerMul {
    private IntegerMul() {}

    public static long mul_i8(long a, long b) {
        return Ints.w8(a * b);
    }

    public static long mul_i16(long a, long b) {
        return Ints.w16(a * b);
    }

    public static long mul_i32(long a, long b) {
        return Ints.w32(a * b);
    }

    public static long mul_i64(long a, long b) {
        return (a * b);
    }

    public static long mul_u8(long a, long b) {
        return Ints.wu8(a * b);
    }

    public static long mul_u16(long a, long b) {
        return Ints.wu16(a * b);
    }

    public static long mul_u32(long a, long b) {
        return Ints.wu32(a * b);
    }

    public static long mul_u64(long a, long b) {
        return (a * b);
    }

}

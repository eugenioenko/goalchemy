package rt;

/** core.integer.neg: wrapping negation. */
public final class IntegerNeg {
    private IntegerNeg() {}

    public static long neg_i8(long a) {
        return Ints.w8(-a);
    }

    public static long neg_i16(long a) {
        return Ints.w16(-a);
    }

    public static long neg_i32(long a) {
        return Ints.w32(-a);
    }

    public static long neg_i64(long a) {
        return (-a);
    }

    public static long neg_u8(long a) {
        return Ints.wu8(-a);
    }

    public static long neg_u16(long a) {
        return Ints.wu16(-a);
    }

    public static long neg_u32(long a) {
        return Ints.wu32(-a);
    }

    public static long neg_u64(long a) {
        return (-a);
    }

}

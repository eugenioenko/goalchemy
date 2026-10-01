package rt;

/** core.integer.rem: remainder with the sign of the dividend. */
public final class IntegerRem {
    private IntegerRem() {}

    public static long rem_i8(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.w8(a % b);
    }

    public static long rem_i16(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.w16(a % b);
    }

    public static long rem_i32(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.w32(a % b);
    }

    public static long rem_i64(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return (a % b);
    }

    public static long rem_u8(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.wu8(a % b);
    }

    public static long rem_u16(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.wu16(a % b);
    }

    public static long rem_u32(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.wu32(a % b);
    }

    public static long rem_u64(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Long.remainderUnsigned(a, b);
    }

}

package rt;

/** core.integer.div: truncating division; a zero divisor panics. */
public final class IntegerDiv {
    private IntegerDiv() {}

    public static long div_i8(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.w8(a / b);
    }

    public static long div_i16(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.w16(a / b);
    }

    public static long div_i32(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.w32(a / b);
    }

    public static long div_i64(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return (a / b);
    }

    public static long div_u8(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.wu8(a / b);
    }

    public static long div_u16(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.wu16(a / b);
    }

    public static long div_u32(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Ints.wu32(a / b);
    }

    public static long div_u64(long a, long b) {
        if (b == 0) throw Ints.divZero();
        return Long.divideUnsigned(a, b);
    }

}

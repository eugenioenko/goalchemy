package rt;

/** core.integer.sub: wrapping subtraction. */
public final class IntegerSub {
    private IntegerSub() {}

    public static long sub_i8(long a, long b) {
        return Ints.w8(a - b);
    }

    public static long sub_i16(long a, long b) {
        return Ints.w16(a - b);
    }

    public static long sub_i32(long a, long b) {
        return Ints.w32(a - b);
    }

    public static long sub_i64(long a, long b) {
        return (a - b);
    }

    public static long sub_u8(long a, long b) {
        return Ints.wu8(a - b);
    }

    public static long sub_u16(long a, long b) {
        return Ints.wu16(a - b);
    }

    public static long sub_u32(long a, long b) {
        return Ints.wu32(a - b);
    }

    public static long sub_u64(long a, long b) {
        return (a - b);
    }

}

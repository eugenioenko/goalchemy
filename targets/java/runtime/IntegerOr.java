package rt;

/** core.integer.or: bitwise OR. */
public final class IntegerOr {
    private IntegerOr() {}

    public static long or_i8(long a, long b) {
        return Ints.w8(a | b);
    }

    public static long or_i16(long a, long b) {
        return Ints.w16(a | b);
    }

    public static long or_i32(long a, long b) {
        return Ints.w32(a | b);
    }

    public static long or_i64(long a, long b) {
        return (a | b);
    }

    public static long or_u8(long a, long b) {
        return Ints.wu8(a | b);
    }

    public static long or_u16(long a, long b) {
        return Ints.wu16(a | b);
    }

    public static long or_u32(long a, long b) {
        return Ints.wu32(a | b);
    }

    public static long or_u64(long a, long b) {
        return (a | b);
    }

}

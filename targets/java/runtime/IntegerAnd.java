package rt;

/** core.integer.and: bitwise AND. */
public final class IntegerAnd {
    private IntegerAnd() {}

    public static long and_i8(long a, long b) {
        return Ints.w8(a & b);
    }

    public static long and_i16(long a, long b) {
        return Ints.w16(a & b);
    }

    public static long and_i32(long a, long b) {
        return Ints.w32(a & b);
    }

    public static long and_i64(long a, long b) {
        return (a & b);
    }

    public static long and_u8(long a, long b) {
        return Ints.wu8(a & b);
    }

    public static long and_u16(long a, long b) {
        return Ints.wu16(a & b);
    }

    public static long and_u32(long a, long b) {
        return Ints.wu32(a & b);
    }

    public static long and_u64(long a, long b) {
        return (a & b);
    }

}

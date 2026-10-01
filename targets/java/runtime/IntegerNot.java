package rt;

/** core.integer.not: bitwise complement. */
public final class IntegerNot {
    private IntegerNot() {}

    public static long not_i8(long a) {
        return Ints.w8(~a);
    }

    public static long not_i16(long a) {
        return Ints.w16(~a);
    }

    public static long not_i32(long a) {
        return Ints.w32(~a);
    }

    public static long not_i64(long a) {
        return (~a);
    }

    public static long not_u8(long a) {
        return Ints.wu8(~a);
    }

    public static long not_u16(long a) {
        return Ints.wu16(~a);
    }

    public static long not_u32(long a) {
        return Ints.wu32(~a);
    }

    public static long not_u64(long a) {
        return (~a);
    }

}

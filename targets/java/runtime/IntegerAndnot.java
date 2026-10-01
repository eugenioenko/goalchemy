package rt;

/** core.integer.andnot: bit clear. */
public final class IntegerAndnot {
    private IntegerAndnot() {}

    public static long andnot_i8(long a, long b) {
        return Ints.w8(a & ~b);
    }

    public static long andnot_i16(long a, long b) {
        return Ints.w16(a & ~b);
    }

    public static long andnot_i32(long a, long b) {
        return Ints.w32(a & ~b);
    }

    public static long andnot_i64(long a, long b) {
        return (a & ~b);
    }

    public static long andnot_u8(long a, long b) {
        return Ints.wu8(a & ~b);
    }

    public static long andnot_u16(long a, long b) {
        return Ints.wu16(a & ~b);
    }

    public static long andnot_u32(long a, long b) {
        return Ints.wu32(a & ~b);
    }

    public static long andnot_u64(long a, long b) {
        return (a & ~b);
    }

}

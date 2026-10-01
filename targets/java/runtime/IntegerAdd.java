package rt;

/** core.integer.add: wrapping addition. */
public final class IntegerAdd {
    private IntegerAdd() {}

    public static long add_i8(long a, long b) {
        return Ints.w8(a + b);
    }

    public static long add_i16(long a, long b) {
        return Ints.w16(a + b);
    }

    public static long add_i32(long a, long b) {
        return Ints.w32(a + b);
    }

    public static long add_i64(long a, long b) {
        return (a + b);
    }

    public static long add_u8(long a, long b) {
        return Ints.wu8(a + b);
    }

    public static long add_u16(long a, long b) {
        return Ints.wu16(a + b);
    }

    public static long add_u32(long a, long b) {
        return Ints.wu32(a + b);
    }

    public static long add_u64(long a, long b) {
        return (a + b);
    }

}

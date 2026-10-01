package rt;

/** core.integer.shl: left shift by a validated count. */
public final class IntegerShl {
    private IntegerShl() {}

    /** n is a validated count (Ints.count or Ints.countu). */
    public static long shl_i8(long a, int n) {
        if (n >= 8) return 0;
        return Ints.w8(a << n);
    }

    public static long shl_i16(long a, int n) {
        if (n >= 16) return 0;
        return Ints.w16(a << n);
    }

    public static long shl_i32(long a, int n) {
        if (n >= 32) return 0;
        return Ints.w32(a << n);
    }

    public static long shl_i64(long a, int n) {
        if (n >= 64) return 0;
        return (a << n);
    }

    public static long shl_u8(long a, int n) {
        if (n >= 8) return 0;
        return Ints.wu8(a << n);
    }

    public static long shl_u16(long a, int n) {
        if (n >= 16) return 0;
        return Ints.wu16(a << n);
    }

    public static long shl_u32(long a, int n) {
        if (n >= 32) return 0;
        return Ints.wu32(a << n);
    }

    public static long shl_u64(long a, int n) {
        if (n >= 64) return 0;
        return (a << n);
    }

}

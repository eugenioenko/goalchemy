package rt;

/** core.integer.xor: bitwise XOR. */
public final class IntegerXor {
    private IntegerXor() {}

    public static long xor_i8(long a, long b) {
        return Ints.w8(a ^ b);
    }

    public static long xor_i16(long a, long b) {
        return Ints.w16(a ^ b);
    }

    public static long xor_i32(long a, long b) {
        return Ints.w32(a ^ b);
    }

    public static long xor_i64(long a, long b) {
        return (a ^ b);
    }

    public static long xor_u8(long a, long b) {
        return Ints.wu8(a ^ b);
    }

    public static long xor_u16(long a, long b) {
        return Ints.wu16(a ^ b);
    }

    public static long xor_u32(long a, long b) {
        return Ints.wu32(a ^ b);
    }

    public static long xor_u64(long a, long b) {
        return (a ^ b);
    }

}

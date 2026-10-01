package rt;

/** core.integer.convert: truncate to the target width and reinterpret signedness. */
public final class IntegerConvert {
    private IntegerConvert() {}

    public static long to_i8(long a) {
        return Ints.w8(a);
    }

    public static long to_i16(long a) {
        return Ints.w16(a);
    }

    public static long to_i32(long a) {
        return Ints.w32(a);
    }

    public static long to_i64(long a) {
        return (a);
    }

    public static long to_u8(long a) {
        return Ints.wu8(a);
    }

    public static long to_u16(long a) {
        return Ints.wu16(a);
    }

    public static long to_u32(long a) {
        return Ints.wu32(a);
    }

    public static long to_u64(long a) {
        return (a);
    }

}

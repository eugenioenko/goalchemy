namespace Rt;

/// <summary>core.integer.and: bitwise and.</summary>
public static partial class R
{
    public static long and_i8(long a, long b) => Ints.w8(a & b);
    public static long and_i16(long a, long b) => Ints.w16(a & b);
    public static long and_i32(long a, long b) => Ints.w32(a & b);
    public static long and_i64(long a, long b) => a & b;
    public static long and_u8(long a, long b) => Ints.wu8(a & b);
    public static long and_u16(long a, long b) => Ints.wu16(a & b);
    public static long and_u32(long a, long b) => Ints.wu32(a & b);
    public static long and_u64(long a, long b) => a & b;
}

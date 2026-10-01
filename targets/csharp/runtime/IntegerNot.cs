namespace Rt;

/// <summary>core.integer.not: bitwise complement.</summary>
public static partial class R
{
    public static long not_i8(long a) => Ints.w8(~a);
    public static long not_i16(long a) => Ints.w16(~a);
    public static long not_i32(long a) => Ints.w32(~a);
    public static long not_i64(long a) => ~a;
    public static long not_u8(long a) => Ints.wu8(~a);
    public static long not_u16(long a) => Ints.wu16(~a);
    public static long not_u32(long a) => Ints.wu32(~a);
    public static long not_u64(long a) => ~a;
}

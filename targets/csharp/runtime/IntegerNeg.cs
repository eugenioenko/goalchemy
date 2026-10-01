namespace Rt;

/// <summary>core.integer.neg: wrapping negation.</summary>
public static partial class R
{
    public static long neg_i8(long a) => Ints.w8(unchecked(-a));
    public static long neg_i16(long a) => Ints.w16(unchecked(-a));
    public static long neg_i32(long a) => Ints.w32(unchecked(-a));
    public static long neg_i64(long a) => unchecked(-a);
    public static long neg_u8(long a) => Ints.wu8(unchecked(-a));
    public static long neg_u16(long a) => Ints.wu16(unchecked(-a));
    public static long neg_u32(long a) => Ints.wu32(unchecked(-a));
    public static long neg_u64(long a) => unchecked(-a);
}

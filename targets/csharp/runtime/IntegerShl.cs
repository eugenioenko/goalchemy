namespace Rt;

/// <summary>core.integer.shl: n is a validated count (Ints.count or Ints.countu).</summary>
public static partial class R
{
    public static long shl_i8(long a, int n) => n >= 8 ? 0 : Ints.w8(a << n);
    public static long shl_i16(long a, int n) => n >= 16 ? 0 : Ints.w16(a << n);
    public static long shl_i32(long a, int n) => n >= 32 ? 0 : Ints.w32(a << n);
    public static long shl_i64(long a, int n) => n >= 64 ? 0 : a << n;
    public static long shl_u8(long a, int n) => n >= 8 ? 0 : Ints.wu8(a << n);
    public static long shl_u16(long a, int n) => n >= 16 ? 0 : Ints.wu16(a << n);
    public static long shl_u32(long a, int n) => n >= 32 ? 0 : Ints.wu32(a << n);
    public static long shl_u64(long a, int n) => n >= 64 ? 0 : a << n;
}

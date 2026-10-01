namespace Rt;

/// <summary>core.integer.shr: arithmetic for signed kinds, logical for unsigned.</summary>
public static partial class R
{
    public static long shr_i8(long a, int n) => n >= 8 ? (a < 0 ? -1 : 0) : a >> n;
    public static long shr_i16(long a, int n) => n >= 16 ? (a < 0 ? -1 : 0) : a >> n;
    public static long shr_i32(long a, int n) => n >= 32 ? (a < 0 ? -1 : 0) : a >> n;
    public static long shr_i64(long a, int n) => n >= 64 ? (a < 0 ? -1 : 0) : a >> n;
    public static long shr_u8(long a, int n) => n >= 8 ? 0 : (long)((ulong)a >> n);
    public static long shr_u16(long a, int n) => n >= 16 ? 0 : (long)((ulong)a >> n);
    public static long shr_u32(long a, int n) => n >= 32 ? 0 : (long)((ulong)a >> n);
    public static long shr_u64(long a, int n) => n >= 64 ? 0 : (long)((ulong)a >> n);
}

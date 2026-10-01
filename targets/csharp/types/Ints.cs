namespace Rt;

/// <summary>Integer kinds: every kind is a long normalized to its Go width;
/// 64-bit unsigned values are stored as their two's-complement bits.</summary>
public static class Ints
{
    public static long w8(long x) => (sbyte)x;
    public static long w16(long x) => (short)x;
    public static long w32(long x) => (int)x;
    public static long w64(long x) => x;
    public static long wu8(long x) => x & 0xFF;
    public static long wu16(long x) => x & 0xFFFF;
    public static long wu32(long x) => x & 0xFFFFFFFFL;
    public static long wu64(long x) => x;

    /// <summary>Validates a signed shift count, clamped to 64.</summary>
    public static int count(long n)
    {
        if (n < 0) throw Panics.runtimePanic("negative shift amount");
        return n > 64 ? 64 : (int)n;
    }

    /// <summary>Validates an unsigned shift count, clamped to 64.</summary>
    public static int countu(long n) => n < 0 || n > 64 ? 64 : (int)n;

    public static Exception divZero() => Panics.runtimePanic("integer divide by zero");

    public static string u64String(long x) => ((ulong)x).ToString();

    public static int cmpu(long a, long b) => ((ulong)a).CompareTo((ulong)b);
}

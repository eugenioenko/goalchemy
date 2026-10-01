namespace Rt;

/// <summary>core.integer.div: truncated division; division by zero panics.</summary>
public static partial class R
{
    public static long div_i8(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return Ints.w8(a / b);
    }

    public static long div_i16(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return Ints.w16(a / b);
    }

    public static long div_i32(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return Ints.w32(a / b);
    }

    public static long div_i64(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        if (b == -1) return unchecked(-a);
        return a / b;
    }

    public static long div_u8(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return Ints.wu8(a / b);
    }

    public static long div_u16(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return Ints.wu16(a / b);
    }

    public static long div_u32(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return Ints.wu32(a / b);
    }

    public static long div_u64(long a, long b)
    {
        if (b == 0) throw Ints.divZero();
        return (long)((ulong)a / (ulong)b);
    }

}

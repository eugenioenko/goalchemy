namespace Rt;

/// <summary>core.slice.make: make([]T, len, cap) with zeroed capacity.</summary>
public static partial class R
{
    public static Slice makeSlice(long len, long cap, Func<object> zero, bool bytes = false)
    {
        if (len < 0 || len > (1L << 53)) throw Panics.runtimePanic("makeslice: len out of range");
        if (cap < len || cap > (1L << 53)) throw Panics.runtimePanic("makeslice: cap out of range");
        if (cap > int.MaxValue - 64) throw Panics.fault("allocation of " + cap + " elements exceeds host limits");
        if (bytes) return new Slice(new byte[(int)cap], 0, (int)len, (int)cap);
        var a = new object[(int)cap];
        for (int i = 0; i < a.Length; i++) a[i] = zero();
        return new Slice(a, 0, (int)len, (int)cap);
    }
}

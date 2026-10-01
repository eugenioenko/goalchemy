package rt;

import java.util.function.Supplier;

/** core.slice.make: make([]T, len, cap) with zeroed capacity. */
public final class SliceMake {
    private SliceMake() {}

    public static Slice makeSlice(long len, long cap, Supplier<Object> zero) {
        if (len < 0 || len > (1L << 53)) throw Panics.runtimePanic("makeslice: len out of range");
        if (cap < len || cap > (1L << 53)) throw Panics.runtimePanic("makeslice: cap out of range");
        if (cap > Integer.MAX_VALUE - 8) throw Panics.fault("allocation of " + cap + " elements exceeds host limits");
        Object[] a = new Object[(int) cap];
        for (int i = 0; i < a.length; i++) a[i] = zero.get();
        return new Slice(a, 0, (int) len, (int) cap);
    }

}

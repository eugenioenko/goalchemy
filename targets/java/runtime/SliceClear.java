package rt;

import java.util.function.Supplier;

/** core.slice.clear: zero the elements below the length. */
public final class SliceClear {
    private SliceClear() {}

    public static void clearSlice(Slice s, Supplier<Object> zero) {
        for (int i = 0; i < s.l; i++) s.a[s.o + i] = zero.get();
    }

}

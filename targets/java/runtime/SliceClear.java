package rt;

import java.util.function.Supplier;

/** core.slice.clear: zero the elements below the length. */
public final class SliceClear {
    private SliceClear() {}

    public static void clearSlice(Slice s, Supplier<Object> zero) {
        if (s.bytes) {
            if (s.l != 0) java.util.Arrays.fill((byte[]) s.a, s.o, s.o + s.l, (byte) 0);
        } else {
            for (int i = 0; i < s.l; i++) s.set(i, zero.get());
        }
    }

}

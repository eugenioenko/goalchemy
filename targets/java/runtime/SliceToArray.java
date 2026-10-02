package rt;

import java.util.function.UnaryOperator;

/** core.slice.to_array: [N]T(s), copying the first N elements. */
public final class SliceToArray {
    private SliceToArray() {}

    public static Object[] sliceToArray(Slice s, int n, UnaryOperator<Object> clone) {
        if (s.l < n) throw Panics.runtimePanic("cannot convert slice with length " + s.l + " to array or pointer to array with length " + n);
        Object[] a = new Object[n];
        for (int i = 0; i < n; i++) a[i] = clone == null ? s.get(i) : clone.apply(s.get(i));
        return a;
    }

    public static byte[] sliceToByteArray(Slice s, int n) {
        if (s.l < n) throw Panics.runtimePanic("cannot convert slice with length " + s.l + " to array or pointer to array with length " + n);
        byte[] a = new byte[n];
        if (n != 0) System.arraycopy(s.a, s.o, a, 0, n);
        return a;
    }
}

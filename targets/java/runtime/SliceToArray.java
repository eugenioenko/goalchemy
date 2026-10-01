package rt;

import java.util.function.UnaryOperator;

/** core.slice.to_array: [N]T(s), copying the first N elements. */
public final class SliceToArray {
    private SliceToArray() {}

    public static Object[] sliceToArray(Slice s, int n, UnaryOperator<Object> clone) {
        if (s.l < n) throw Panics.runtimePanic("cannot convert slice with length " + s.l + " to array or pointer to array with length " + n);
        Object[] a = new Object[n];
        for (int i = 0; i < n; i++) a[i] = clone == null ? s.a[s.o + i] : clone.apply(s.a[s.o + i]);
        return a;
    }

}

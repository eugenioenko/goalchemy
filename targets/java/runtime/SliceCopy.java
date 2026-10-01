package rt;

import java.util.function.UnaryOperator;

/** core.slice.copy: copy(dst, src) through a temporary for overlap safety. */
public final class SliceCopy {
    private SliceCopy() {}

    public static long copy(Slice dst, Slice src, UnaryOperator<Object> clone) {
        int n = Math.min(dst.l, src.l);
        if (n == 0) return 0;
        Object[] tmp = new Object[n];
        System.arraycopy(src.a, src.o, tmp, 0, n);
        for (int i = 0; i < n; i++) dst.a[dst.o + i] = clone == null ? tmp[i] : clone.apply(tmp[i]);
        return n;
    }

    public static long copyString(Slice dst, String s) {
        int n = Math.min(dst.l, s.length());
        for (int i = 0; i < n; i++) dst.a[dst.o + i] = (long) s.charAt(i);
        return n;
    }

}

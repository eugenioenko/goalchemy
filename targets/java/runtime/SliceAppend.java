package rt;

import java.util.function.UnaryOperator;

/** core.slice.append: append with Goalchemy's growth rule; aggregate elements are cloned when they move. */
public final class SliceAppend {
    private SliceAppend() {}

    public static long growCap(long old, long required) {
        long doubled = old <= Long.MAX_VALUE / 2 ? 2 * old : Long.MAX_VALUE;
        return Math.max(required, Math.max(1, doubled));
    }

    private static Slice values(Slice s, Object[] vs, UnaryOperator<Object> clone) {
        int n = s.l + vs.length;
        if (vs.length == 0) return s;
        if (n <= s.c) {
            System.arraycopy(vs, 0, s.a, s.o + s.l, vs.length);
            return new Slice(s.a, s.o, n, s.c);
        }
        long c = growCap(s.c, n);
        if (c > Integer.MAX_VALUE - 8) throw Panics.fault("slice growth to " + c + " elements exceeds host limits");
        Object[] a = new Object[(int) c];
        for (int i = 0; i < s.l; i++) a[i] = clone == null ? s.a[s.o + i] : clone.apply(s.a[s.o + i]);
        System.arraycopy(vs, 0, a, s.l, vs.length);
        return new Slice(a, 0, n, (int) c);
    }

    public static Slice append(Slice s, Object[] vs, UnaryOperator<Object> clone) {
        return values(s, vs, clone);
    }

    public static Slice appendSlice(Slice s, Slice t, UnaryOperator<Object> clone) {
        if (t.l == 0) return s.a == null && t.a == null ? Slice.NIL : s;
        Object[] vs = new Object[t.l];
        for (int i = 0; i < t.l; i++) vs[i] = clone == null ? t.a[t.o + i] : clone.apply(t.a[t.o + i]);
        return values(s, vs, clone);
    }

    public static Slice appendString(Slice b, String s) {
        Object[] vs = new Object[s.length()];
        for (int i = 0; i < vs.length; i++) vs[i] = (long) s.charAt(i);
        return values(b, vs, null);
    }

}

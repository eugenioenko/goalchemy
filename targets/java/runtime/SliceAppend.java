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
        long required = (long) s.l + vs.length;
        if (required > Integer.MAX_VALUE - 8) throw Panics.fault("slice growth to " + required + " elements exceeds host limits");
        int n = (int) required;
        if (vs.length == 0) return s;
        if (n <= s.c) {
            System.arraycopy(vs, 0, s.a, s.o + s.l, vs.length);
            return new Slice(s.a, s.o, n, s.c);
        }
        long c = growCap(s.c, n);
        if (c > Integer.MAX_VALUE - 8) throw Panics.fault("slice growth to " + c + " elements exceeds host limits");
        Object[] a = new Object[(int) c];
        for (int i = 0; i < s.l; i++) a[i] = clone == null ? s.get(i) : clone.apply(s.get(i));
        System.arraycopy(vs, 0, a, s.l, vs.length);
        return new Slice(a, 0, n, (int) c);
    }

    public static Slice append(Slice s, Object[] vs, UnaryOperator<Object> clone) {
        return values(s, vs, clone);
    }

    public static Slice appendSlice(Slice s, Slice t, UnaryOperator<Object> clone) {
        if (s.bytes) return appendBytes(s, (byte[]) t.a, t.o, t.l);
        if (t.l == 0) return s;
        Object[] vs = new Object[t.l];
        for (int i = 0; i < t.l; i++) vs[i] = clone == null ? t.get(i) : clone.apply(t.get(i));
        return values(s, vs, clone);
    }

    public static Slice appendString(Slice b, String s) {
        if (b.bytes) {
            Slice r = byteRoom(b, s.length());
            for (int i = 0; i < s.length(); i++) ((byte[]) r.a)[r.o + b.l + i] = (byte) s.charAt(i);
            return r;
        }
        Object[] vs = new Object[s.length()];
        for (int i = 0; i < vs.length; i++) vs[i] = (long) s.charAt(i);
        return values(b, vs, null);
    }

    /** Allocate before writing; both addition and growth use wide arithmetic. */
    private static Slice byteRoom(Slice s, int count) {
        if (count == 0) return s;
        long required = (long) s.l + count;
        if (required > Integer.MAX_VALUE - 8) throw Panics.fault("slice growth to " + required + " elements exceeds host limits");
        int n = (int) required;
        if (n <= s.c) return new Slice(s.a, s.o, n, s.c, true);
        long cap = growCap(s.c, required);
        if (cap > Integer.MAX_VALUE - 8) throw Panics.fault("slice growth to " + cap + " elements exceeds host limits");
        byte[] a = new byte[(int) cap];
        if (s.l != 0) System.arraycopy(s.a, s.o, a, 0, s.l);
        return new Slice(a, 0, n, (int) cap);
    }

    public static Slice appendBytes(Slice s, byte[] values) {
        return appendBytes(s, values, 0, values.length);
    }

    private static Slice appendBytes(Slice s, byte[] values, int offset, int count) {
        Slice r = byteRoom(s, count);
        if (count != 0) System.arraycopy(values, offset, r.a, r.o + s.l, count);
        return r;
    }

}

package rt;

/** core.slice.slice: s[lo:hi:max] sharing backing storage; null bounds use defaults. */
public final class SliceSlice {
    private SliceSlice() {}

    public static Slice reslice(Slice s, Long lo, Long hi, Long max, boolean u) {
        long l = lo == null ? 0 : lo;
        long h = hi == null ? s.l : hi;
        long m = s.c;
        if (max != null) {
            Bounds.check3(l, h, max, s.c, "capacity", u);
            m = max;
        } else {
            Bounds.check2(l, h, s.c, "capacity", u);
        }
        if (s.a == null) return s;
        return new Slice(s.a, s.o + (int) l, (int) (h - l), (int) (m - l), s.bytes);
    }

    /** Slices an array through a pointer: (&a)[lo:hi:max]. */
    public static Slice sliceArray(Object[] a, Long lo, Long hi, Long max, boolean u) {
        long l = lo == null ? 0 : lo;
        long h = hi == null ? a.length : hi;
        long m = a.length;
        if (max != null) {
            Bounds.check3(l, h, max, a.length, "length", u);
            m = max;
        } else {
            Bounds.check2(l, h, a.length, "length", u);
        }
        return new Slice(a, (int) l, (int) (h - l), (int) (m - l));
    }

    /** Slices an array through a pointer: (&a)[lo:hi:max]. */
    public static Slice sliceArray(byte[] a, Long lo, Long hi, Long max, boolean u) {
        long l = lo == null ? 0 : lo;
        long h = hi == null ? a.length : hi;
        long m = a.length;
        if (max != null) {
            Bounds.check3(l, h, max, a.length, "length", u);
            m = max;
        } else {
            Bounds.check2(l, h, a.length, "length", u);
        }
        return new Slice(a, (int) l, (int) (h - l), (int) (m - l));
    }

}

package rt;

/** Immutable slice headers over shared Object[] or native byte[] storage. */
public final class Slice {
    public final Object a;
    public final int o;
    public final int l;
    public final int c;
    public final boolean bytes;

    public Slice(Object a, int o, int l, int c) {
        this(a, o, l, c, a instanceof byte[]);
    }

    public Slice(Object a, int o, int l, int c, boolean bytes) {
        this.a = a;
        this.o = o;
        this.l = l;
        this.c = c;
        this.bytes = bytes;
    }

    /** Generic boundary access; byte hot paths use primitive storage directly. */
    public Object get(int i) {
        return bytes ? (long) (((byte[]) a)[o + i] & 255) : ((Object[]) a)[o + i];
    }

    public void set(int i, Object v) {
        if (bytes) ((byte[]) a)[o + i] = (byte) (long) (Long) v;
        else ((Object[]) a)[o + i] = v;
    }

    public static final Slice NIL = new Slice(null, 0, 0, 0, false);
    public static final Slice BYTE_NIL = new Slice(null, 0, 0, 0, true);
}

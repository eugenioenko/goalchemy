package rt;

/** Slice headers over shared backing arrays. Headers are immutable values. */
public final class Slice {
    public final Object[] a;
    public final int o;
    public final int l;
    public final int c;

    public Slice(Object[] a, int o, int l, int c) {
        this.a = a;
        this.o = o;
        this.l = l;
        this.c = c;
    }

    public static final Slice NIL = new Slice(null, 0, 0, 0);
}

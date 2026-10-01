package rt;

/** Heap storage for a boxed variable or new(T). */
public final class Cell implements Ref {
    public Object v;

    public Cell(Object v) {
        this.v = v;
    }

    public Object get() {
        return v;
    }

    public void set(Object x) {
        v = x;
    }
}

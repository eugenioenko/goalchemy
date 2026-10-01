package rt;

/** A source panic carrying a Go interface value. */
public final class GoPanic extends RuntimeException {
    public final Box value;
    public boolean recovered;
    public GoPanic prev;

    public GoPanic(Box value) {
        super("go panic", null, false, false);
        this.value = value;
    }
}

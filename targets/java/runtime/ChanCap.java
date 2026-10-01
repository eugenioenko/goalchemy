package rt;

/** core.chan.cap: buffer capacity. */
public final class ChanCap {
    private ChanCap() {}

    public static long chanCap(ChanMake.Chan ch) {
        return ch == null ? 0 : ch.size;
    }
}

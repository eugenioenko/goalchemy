package rt;

/** core.chan.len: buffered element count. */
public final class ChanLen {
    private ChanLen() {}

    public static long chanLen(ChanMake.Chan ch) {
        return ch == null ? 0 : ch.buf.size();
    }
}

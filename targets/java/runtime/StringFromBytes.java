package rt;

/** core.string.from_bytes: string(b), copying the bytes. */
public final class StringFromBytes {
    private StringFromBytes() {}

    public static String fromBytes(Slice b) {
        if (b.a == null) return "";
        StringBuilder sb = new StringBuilder(b.l);
        for (int i = 0; i < b.l; i++) sb.append((char) (long) (Long) b.a[b.o + i]);
        return sb.toString();
    }

}

package rt;

/** core.string.to_bytes: []byte(s) with capacity equal to length. */
public final class StringToBytes {
    private StringToBytes() {}

    public static Slice toBytes(String s) {
        Object[] a = new Object[s.length()];
        for (int i = 0; i < a.length; i++) a[i] = (long) s.charAt(i);
        return new Slice(a, 0, a.length, a.length);
    }

}

package rt;

/** core.string.to_runes: []rune(s) with capacity equal to length. */
public final class StringToRunes {
    private StringToRunes() {}

    public static Slice toRunes(String s) {
        java.util.ArrayList<Object> a = new java.util.ArrayList<>();
        for (int i = 0; i < s.length(); ) {
            int[] r = Utf8.decode(s, i);
            a.add((long) r[0]);
            i += r[1];
        }
        Object[] arr = a.toArray();
        return new Slice(arr, 0, arr.length, arr.length);
    }

}

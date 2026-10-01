package rt;

/** core.string.from_runes: string(r) for a rune slice. */
public final class StringFromRunes {
    private StringFromRunes() {}

    public static String fromRunes(Slice r) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < r.l; i++) sb.append(Utf8.encode((Long) r.a[r.o + i]));
        return sb.toString();
    }

}

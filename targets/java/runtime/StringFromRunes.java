package rt;

/** core.string.from_runes: string(r) for a rune slice. */
public final class StringFromRunes {
    private StringFromRunes() {}

    public static String fromRunes(Slice r) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < r.l; i++) sb.append(Utf8.encode((Long) r.get(i)));
        return sb.toString();
    }

}

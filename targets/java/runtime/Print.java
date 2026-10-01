package rt;

/** core.print: print and println of Booleans, integers, and strings. */
public final class Print {
    private Print() {}

    /** Operands are Boolean, Long, or String; unsigned 64-bit operands are
     * formatted by the caller. */
    public static String printString(Object v) {
        return v instanceof String s ? s : String.valueOf(v);
    }

    public static String printString(Object v, boolean unsigned) {
        return unsigned ? Long.toUnsignedString((Long) v) : printString(v);
    }

    public static void print(boolean newline, Object... args) {
        StringBuilder b = new StringBuilder();
        for (int i = 0; i < args.length; i++) {
            if (newline && i > 0) b.append(' ');
            b.append(printString(args[i]));
        }
        if (newline) b.append('\n');
        Out.stderr(b.toString());
    }

}

package rt;

/** Runtime errors, faults, and checks shared by the runtime functions. */
public final class Panics {
    private Panics() {}

    static TypeDesc errorType(String name, String prefix) {
        return new TypeDesc(name, "runtime_error", (a, b) -> a.equals(b), a -> a,
                TypeDesc.methods("Error", (Fn) a -> prefix + a[0], "RuntimeError", (Fn) a -> null), null, true);
    }

    public static final TypeDesc RUNTIME_ERROR = errorType("runtime.Error", "runtime error: ");
    public static final TypeDesc PLAIN_ERROR = errorType("runtime.plainError", "");
    public static final TypeDesc TYPE_ASSERTION_ERROR = errorType("*runtime.TypeAssertionError", "");

    public static GoPanic runtimePanic(String msg) {
        return new GoPanic(new Box(RUNTIME_ERROR, msg));
    }

    public static GoPanic plainPanic(String msg) {
        return new GoPanic(new Box(PLAIN_ERROR, msg));
    }

    public static GoPanic uncomparable(String name) {
        return runtimePanic("comparing uncomparable type " + name);
    }

    public static GoPanic unhashable(String name) {
        return runtimePanic("hash of unhashable type " + name);
    }

    public static boolean uncomparableEq(String name) {
        throw uncomparable(name);
    }

    public static Object unhashableKey(String name) {
        throw unhashable(name);
    }

    public static GoPanic nilDeref() {
        return runtimePanic("invalid memory address or nil pointer dereference");
    }

    public static <T> T nilchk(T p) {
        if (p == null) throw nilDeref();
        return p;
    }

    /** An implementation fault: never a source panic. */
    public static RuntimeException fault(String msg) {
        return new IllegalStateException("goalchemy fault: " + msg);
    }

    /** Checks a signed index against a length. */
    public static int idx(long i, int len) {
        if (i < 0) throw runtimePanic("index out of range [" + i + "]");
        if (i >= len) throw runtimePanic("index out of range [" + i + "] with length " + len);
        return (int) i;
    }

    /** Checks an unsigned 64-bit index against a length. */
    public static int idxu(long i, int len) {
        if (i < 0 || i >= len) throw runtimePanic("index out of range [" + Long.toUnsignedString(i) + "] with length " + len);
        return (int) i;
    }
}

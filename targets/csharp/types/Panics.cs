namespace Rt;

/// <summary>Runtime errors, faults, and checks shared by the runtime functions.</summary>
public static class Panics
{
    static TypeDesc errorType(string name, string prefix) =>
        new TypeDesc(name, "runtime_error", (a, b) => Equals(a, b), a => a,
            TypeDesc.MethodsOf("Error", new Fn(a => prefix + (string)a[0]), "RuntimeError", new Fn(a => null)), null, true);

    public static readonly TypeDesc RUNTIME_ERROR = errorType("runtime.Error", "runtime error: ");
    public static readonly TypeDesc PLAIN_ERROR = errorType("runtime.plainError", "");
    public static readonly TypeDesc TYPE_ASSERTION_ERROR = errorType("*runtime.TypeAssertionError", "");

    public static GoPanic runtimePanic(string msg) => new GoPanic(new Box(RUNTIME_ERROR, msg));

    public static GoPanic plainPanic(string msg) => new GoPanic(new Box(PLAIN_ERROR, msg));

    public static GoPanic uncomparable(string name) => runtimePanic("comparing uncomparable type " + name);

    public static GoPanic unhashable(string name) => runtimePanic("hash of unhashable type " + name);

    public static bool uncomparableEq(string name) => throw uncomparable(name);

    public static object unhashableKey(string name) => throw unhashable(name);

    public static GoPanic nilDeref() => runtimePanic("invalid memory address or nil pointer dereference");

    public static T nilchk<T>(T p) where T : class
    {
        if (p == null) throw nilDeref();
        return p;
    }

    public static Exception fault(string msg) => new Fault(msg);

    /// <summary>Checks a signed index against a length.</summary>
    public static int idx(long i, int len)
    {
        if (i < 0) throw runtimePanic("index out of range [" + i + "]");
        if (i >= len) throw runtimePanic("index out of range [" + i + "] with length " + len);
        return (int)i;
    }

    /// <summary>Checks an unsigned 64-bit index against a length.</summary>
    public static int idxu(long i, int len)
    {
        if (i < 0 || i >= len) throw runtimePanic("index out of range [" + ((ulong)i) + "] with length " + len);
        return (int)i;
    }
}

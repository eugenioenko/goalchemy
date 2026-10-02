namespace Rt;

/// <summary>core.slice.to_array: [N]T(s) copies the first N elements.</summary>
public static partial class R
{
    public static object[] sliceToArray(Slice s, int n, Func<object, object> clone)
    {
        if (s.l < n) throw Panics.runtimePanic("cannot convert slice with length " + s.l + " to array or pointer to array with length " + n);
        var a = new object[n];
        for (int i = 0; i < n; i++) a[i] = clone == null ? s.Get(i) : clone(s.Get(i));
        return a;
    }
    public static byte[] sliceToByteArray(Slice s, int n)
    {
        if (s.l < n) throw Panics.runtimePanic("cannot convert slice with length " + s.l + " to array or pointer to array with length " + n);
        var a = new byte[n];
        if (n != 0) Array.Copy(s.a, s.o, a, 0, n);
        return a;
    }
}

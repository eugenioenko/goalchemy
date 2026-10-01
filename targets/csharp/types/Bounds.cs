namespace Rt;

/// <summary>Go slice-bounds checks and messages. When u is set the bounds
/// have an unsigned 64-bit type: negative longs are huge values.</summary>
public static class Bounds
{
    static Exception oob(string m) => Panics.runtimePanic("slice bounds out of range " + m);

    static string s(long x, bool u) => u ? ((ulong)x).ToString() : x.ToString();

    static bool neg(long x, bool u) => !u && x < 0;

    static bool gt(long x, long y, bool u) => u ? (ulong)x > (ulong)y : x > y;

    public static void check2(long lo, long hi, long limit, string word, bool u)
    {
        if (neg(hi, u)) throw oob("[:" + s(hi, u) + "]");
        if (gt(hi, limit, u)) throw oob("[:" + s(hi, u) + "] with " + word + " " + limit);
        if (neg(lo, u)) throw oob("[" + s(lo, u) + ":]");
        if (gt(lo, hi, u)) throw oob("[" + s(lo, u) + ":" + s(hi, u) + "]");
    }

    public static void check3(long lo, long hi, long max, long limit, string word, bool u)
    {
        if (neg(max, u)) throw oob("[::" + s(max, u) + "]");
        if (gt(max, limit, u)) throw oob("[::" + s(max, u) + "] with " + word + " " + limit);
        if (neg(hi, u)) throw oob("[:" + s(hi, u) + ":]");
        if (gt(hi, max, u)) throw oob("[:" + s(hi, u) + ":" + s(max, u) + "]");
        if (neg(lo, u)) throw oob("[" + s(lo, u) + "::]");
        if (gt(lo, hi, u)) throw oob("[" + s(lo, u) + ":" + s(hi, u) + ":]");
    }
}

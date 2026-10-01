namespace Rt;

/// <summary>core.string.decode_rune: the rune at a byte offset and its width.</summary>
public static partial class R
{
    public static object[] decodeRune(string s, long i)
    {
        var (r, w) = Utf8.decode(s, (int)i);
        return new object[] { (long)r, (long)w };
    }
}

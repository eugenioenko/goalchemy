namespace Rt;

/// <summary>core.string.from_rune: string(r) encodes one code point.</summary>
public static partial class R
{
    public static string fromRune(long x) => Utf8.encode(x);

    /// <summary>For unsigned 64-bit operands: negative longs are huge values.</summary>
    public static string fromRuneU(long x) => Utf8.encode(x < 0 ? -1 : x);
}

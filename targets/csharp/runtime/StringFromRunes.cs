namespace Rt;

/// <summary>core.string.from_runes: string(runes) encodes each code point.</summary>
public static partial class R
{
    public static string fromRunes(Slice r)
    {
        var sb = new System.Text.StringBuilder();
        for (int i = 0; i < r.l; i++) sb.Append(Utf8.encode((long)r.Get(i)));
        return sb.ToString();
    }
}

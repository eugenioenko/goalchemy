namespace Rt;

/// <summary>core.string.to_runes: []rune(s) with capacity equal to length.</summary>
public static partial class R
{
    public static Slice toRunes(string s)
    {
        var a = new List<object>();
        for (int i = 0; i < s.Length;)
        {
            var (r, w) = Utf8.decode(s, i);
            a.Add((long)r);
            i += w;
        }
        var arr = a.ToArray();
        return new Slice(arr, 0, arr.Length, arr.Length);
    }
}

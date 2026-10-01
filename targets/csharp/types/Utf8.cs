namespace Rt;

/// <summary>UTF-8 over binary strings (one char per byte).</summary>
public static class Utf8
{
    public const int RUNE_ERROR = 0xFFFD;

    /// <summary>Decodes the rune at byte offset i: (rune, width).</summary>
    public static (int, int) decode(string s, int i)
    {
        int n = s.Length - i;
        int b0 = s[i];
        if (b0 < 0x80) return (b0, 1);
        if (b0 >= 0xC2 && b0 <= 0xDF)
        {
            if (n < 2) return err();
            int b1 = s[i + 1];
            if (b1 < 0x80 || b1 > 0xBF) return err();
            return (((b0 & 0x1F) << 6) | (b1 & 0x3F), 2);
        }
        if (b0 >= 0xE0 && b0 <= 0xEF)
        {
            if (n < 3) return err();
            int b1 = s[i + 1];
            int lo = b0 == 0xE0 ? 0xA0 : 0x80;
            int hi = b0 == 0xED ? 0x9F : 0xBF;
            if (b1 < lo || b1 > hi) return err();
            int b2 = s[i + 2];
            if (b2 < 0x80 || b2 > 0xBF) return err();
            return (((b0 & 0x0F) << 12) | ((b1 & 0x3F) << 6) | (b2 & 0x3F), 3);
        }
        if (b0 >= 0xF0 && b0 <= 0xF4)
        {
            if (n < 4) return err();
            int b1 = s[i + 1];
            int lo = b0 == 0xF0 ? 0x90 : 0x80;
            int hi = b0 == 0xF4 ? 0x8F : 0xBF;
            if (b1 < lo || b1 > hi) return err();
            int b2 = s[i + 2];
            int b3 = s[i + 3];
            if (b2 < 0x80 || b2 > 0xBF || b3 < 0x80 || b3 > 0xBF) return err();
            return (((b0 & 0x07) << 18) | ((b1 & 0x3F) << 12) | ((b2 & 0x3F) << 6) | (b3 & 0x3F), 4);
        }
        return err();
    }

    static (int, int) err() => (RUNE_ERROR, 1);

    /// <summary>Encodes a code point; invalid code points encode U+FFFD.</summary>
    public static string encode(long r)
    {
        if (r < 0 || r > 0x10FFFF || (r >= 0xD800 && r <= 0xDFFF)) r = RUNE_ERROR;
        int c = (int)r;
        if (c < 0x80) return ((char)c).ToString();
        if (c < 0x800) return new string(new[] { (char)(0xC0 | (c >> 6)), (char)(0x80 | (c & 0x3F)) });
        if (c < 0x10000)
            return new string(new[] { (char)(0xE0 | (c >> 12)), (char)(0x80 | ((c >> 6) & 0x3F)), (char)(0x80 | (c & 0x3F)) });
        return new string(new[] { (char)(0xF0 | (c >> 18)), (char)(0x80 | ((c >> 12) & 0x3F)),
            (char)(0x80 | ((c >> 6) & 0x3F)), (char)(0x80 | (c & 0x3F)) });
    }
}

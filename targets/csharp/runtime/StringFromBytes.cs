namespace Rt;

/// <summary>core.string.from_bytes: string(b) copies the bytes.</summary>
public static partial class R
{
    public static string fromBytes(Slice b)
    {
        if (b.a == null) return "";
        var cs = new char[b.l];
        for (int i = 0; i < b.l; i++) cs[i] = (char)((byte[])b.a)[b.o + i];
        return new string(cs);
    }
}

namespace Rt;

/// <summary>Byte-exact output to standard error and standard output.</summary>
public static class Out
{
    static readonly Stream err = Console.OpenStandardError();
    static readonly Stream @out = Console.OpenStandardOutput();

    public static void stderr(string s)
    {
        var b = System.Text.Encoding.Latin1.GetBytes(s);
        err.Write(b, 0, b.Length);
        err.Flush();
    }

    public static void stdout(string s)
    {
        var b = System.Text.Encoding.Latin1.GetBytes(s);
        @out.Write(b, 0, b.Length);
        @out.Flush();
    }
}

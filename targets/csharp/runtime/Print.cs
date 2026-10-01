namespace Rt;

/// <summary>core.print: print and println of Booleans, integers, and strings.
/// Unsigned 64-bit operands are formatted by the caller.</summary>
public static partial class R
{
    public static string printString(object v) => v switch
    {
        string s => s,
        bool b => b ? "true" : "false",
        _ => Convert.ToString(v, System.Globalization.CultureInfo.InvariantCulture),
    };

    public static string printString(object v, bool unsigned) => unsigned ? ((ulong)(long)v).ToString() : printString(v);

    public static void print(bool newline, params object[] args)
    {
        var b = new System.Text.StringBuilder();
        for (int i = 0; i < args.Length; i++)
        {
            if (newline && i > 0) b.Append(' ');
            b.Append(printString(args[i]));
        }
        if (newline) b.Append('\n');
        Out.stderr(b.ToString());
    }
}

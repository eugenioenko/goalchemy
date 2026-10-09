namespace Rt;

/// <summary>Host log sink for std/log/slog. Levels follow log/slog: -4 debug, 0 info,
/// 4 warn, 8 error. Without a handler, records at warn and above go to standard error.</summary>
public static class Log
{
    /// <summary>One record; Attrs are key/value pairs with group names joined to keys by ".".</summary>
    public sealed record Record(int Level, long UnixNano, string Message, IReadOnlyList<KeyValuePair<string, string>> Attrs, string Text)
    {
        public DateTimeOffset Time => DateTimeOffset.UnixEpoch.AddTicks(UnixNano / 100);
    }

    sealed class Sink { public readonly Action<Record> Handler; public readonly long Level; public Sink(Action<Record> handler, long level) { Handler = handler; Level = level; } }

    static volatile Sink sink = new(null, 4);

    /// <summary>Routes records at level and above to handler; null restores standard error.</summary>
    public static void SetHandler(Action<Record> handler, int level = 4) => sink = new Sink(handler, level);

    /// <summary>A handler that writes each record's text to System.Diagnostics.Trace.</summary>
    public static Action<Record> TraceHandler() => r =>
    {
        if (r.Level >= 8) System.Diagnostics.Trace.TraceError(r.Text);
        else if (r.Level >= 4) System.Diagnostics.Trace.TraceWarning(r.Text);
        else System.Diagnostics.Trace.TraceInformation(r.Text);
    };

    public static bool Enabled(long level) => level >= sink.Level;

    static string Text(string binary) => System.Text.Encoding.UTF8.GetString(System.Text.Encoding.Latin1.GetBytes(binary));

    public static void Emit(long level, long unixNano, string message, Slice attrs, string line)
    {
        var handler = sink.Handler;
        if (handler == null) { Out.stderr(line + "\n"); return; }
        var pairs = new List<KeyValuePair<string, string>>();
        for (int i = 0; i + 1 < attrs.l; i += 2) pairs.Add(new(Text((string)attrs.Get(i)), Text((string)attrs.Get(i + 1))));
        try { handler(new Record((int)Math.Clamp(level, int.MinValue, int.MaxValue), unixNano, Text(message), pairs.AsReadOnly(), Text(line))); } catch (Exception) { }
    }
}

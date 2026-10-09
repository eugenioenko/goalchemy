using System;
using System.Collections.Generic;
using System.Linq;
using Rt;
using Generated = Goalchemy.Generated.GoProgram;

static class LogConsumer
{
    static void Check(bool value, string label) { if (!value) throw new Exception(label); }
    static T Get<T>(Library.Operation<T> operation) => operation.Completion.WaitAsync(TimeSpan.FromSeconds(20)).GetAwaiter().GetResult();
    static void Main()
    {
        var got = new List<Log.Record>();
        Log.SetHandler(got.Add, -4);
        Check(Get(Generated.Work(3, null)) == 6, "result");
        Check(got.Count == 4, "record count " + got.Count);
        Check(got[0].Level == -4 && got[0].Text == "level=DEBUG msg=start sdk=probe n=3", "debug record");
        Check(got[1].Text == "level=INFO msg=info sdk=probe unicode=\"héllo wörld\"" && got[1].Attrs[1].Value == "héllo wörld", "utf8 decoding");
        Check(got[2].Level == 4 && got[2].Message == "retry" && got[2].Attrs[1].Key == "kas.url" && got[2].Attrs[1].Value == "https://kas", "group attrs");
        Check(got[3].Text == "level=ERROR msg=failed err=boom" && Math.Abs((got[3].Time - DateTimeOffset.UtcNow).TotalSeconds) < 60, "error record");
        got.Clear();
        Log.SetHandler(got.Add, 4);
        Get(Generated.Work(1, null));
        Check(got.Count == 2 && got[0].Message == "retry" && got[1].Level == 8, "host level");
        Log.SetHandler(_ => throw new InvalidOperationException("sink"), -4);
        Check(Get(Generated.Work(1, null)) == 2, "sink errors are discarded");
        Log.SetHandler(null);
        Console.WriteLine("PASS log library");
    }
}

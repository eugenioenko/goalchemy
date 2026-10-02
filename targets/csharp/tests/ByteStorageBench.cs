using System;
using System.Diagnostics;
using System.Globalization;
using Rt;

// Fresh-process diagnostic. Mandatory tests assert storage, not noisy heap thresholds.
public static class ByteStorageBench
{
    static Slice[] retained;
    static long Used() => GC.GetTotalMemory(true);

    public static void Main(string[] args)
    {
        bool bytes = args[0] == "native";
        if (!bytes && args[0] != "boxed") throw new ArgumentException("native|boxed MiB");
        int n = checked(int.Parse(args[1], CultureInfo.InvariantCulture) * 1024 * 1024);
        for (int pass = 0; pass < 2000; pass++)
        {
            Slice warm = R.makeSlice(256, 256, () => 0L, bytes);
            R.copy(warm, warm, null);
            R.appendSlice(warm, warm, null);
        }
        long before = Used();
        Slice src = R.makeSlice(n, n, () => 0L, bytes);
        Slice dst = R.makeSlice(n, n, () => 0L, bytes);
        if (bytes)
        {
            byte[] a = (byte[])src.a;
            for (int i = 0; i < n; i++) a[i] = unchecked((byte)i);
        }
        else
        {
            object[] a = (object[])src.a;
            for (int i = 0; i < n; i++) a[i] = (long)(i & 255);
        }
        var timer = Stopwatch.StartNew();
        for (int i = 0; i < 16; i++) R.copy(dst, src, null);
        double copyTime = timer.Elapsed.TotalMilliseconds;
        timer.Restart();
        Slice appended = R.appendSlice(src, src, null);
        double appendTime = timer.Elapsed.TotalMilliseconds;
        retained = new Slice[] { src, dst, appended };
        long delta = Used() - before;
        long backing = 0;
        if (bytes)
        {
            foreach (Slice s in retained) backing += ((byte[])s.a).LongLength;
            if (backing != 4L * n) throw new Exception("retained byte backing size");
        }
        foreach (Slice s in retained)
            if (!s.Get(0).Equals(0L) || !s.Get(s.l - 1).Equals(255L)) throw new Exception("payload reachability");
        Console.WriteLine(string.Format(CultureInfo.InvariantCulture,
            "mode={0} input={1} retained_backing_bytes={2} heap_delta={3} copy16_ms={4:F3} append_ms={5:F3}",
            args[0], n, bytes ? backing.ToString(CultureInfo.InvariantCulture) : "boxed", delta, copyTime, appendTime));
        GC.KeepAlive(retained);
    }
}

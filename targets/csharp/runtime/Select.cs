namespace Rt;

/// <summary>core.select: chooses uniformly among ready cases with the shared
/// choice source; rv is {index, value, ok} with index -1 for default.</summary>
public static partial class R
{
    public static object[] scase(Chan ch, bool send, object v) => new object[] { ch, send, v };

    public static void select(GoTask t, bool hasDefault, params object[][] cases)
    {
        var ready = new List<int>();
        for (int i = 0; i < cases.Length; i++)
        {
            var ch = (Chan)cases[i][0];
            if (ch == null) continue;
            if ((bool)cases[i][1])
            {
                if (ch.closed || hasLive(ch.recvq) || ch.buf.Count < ch.size) ready.Add(i);
            }
            else if (ch.buf.Count > 0 || hasLive(ch.sendq) || ch.closed)
            {
                ready.Add(i);
            }
        }
        if (ready.Count > 0)
        {
            int i = ready[sched.choose(ready.Count)];
            var ch = (Chan)cases[i][0];
            if ((bool)cases[i][1])
            {
                if (ch.closed) throw Panics.plainPanic("send on closed channel");
                var w = dequeue(ch.recvq);
                if (w != null) w.recvDone(cases[i][2], true);
                else ch.buf.Enqueue(cases[i][2]);
                t.rv = new object[] { (long)i, null, false };
                return;
            }
            var r = tryRecv(ch);
            t.rv = new object[] { (long)i, r[0], r[1] };
            return;
        }
        if (hasDefault)
        {
            t.rv = new object[] { -1L, null, false };
            return;
        }
        var st = new SelectState();
        for (int i = 0; i < cases.Length; i++)
        {
            var ch = (Chan)cases[i][0];
            if (ch == null) continue;
            bool send = (bool)cases[i][1];
            var w = new Waiter(t, send ? cases[i][2] : null, st, i);
            (send ? ch.sendq : ch.recvq).Add(w);
        }
        sched.block(t);
        t.cleanup = () =>
        {
            foreach (var c in cases)
            {
                if (c[0] is Chan ch)
                {
                    ch.sendq.RemoveAll(w => w.sel == st);
                    ch.recvq.RemoveAll(w => w.sel == st);
                }
            }
        };
    }
}

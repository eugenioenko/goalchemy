namespace Rt;

/// <summary>core.chan.send: a pause primitive.</summary>
public static partial class R
{
    public static void chanSend(GoTask t, Chan ch, object v)
    {
        t.rv = Array.Empty<object>();
        if (ch == null)
        {
            sched.block(t);
            return;
        }
        if (ch.closed) throw Panics.plainPanic("send on closed channel");
        var r = dequeue(ch.recvq);
        if (r != null)
        {
            r.recvDone(v, true);
            return;
        }
        if (ch.buf.Count < ch.size)
        {
            ch.buf.Enqueue(v);
            return;
        }
        ch.sendq.Add(new Waiter(t, v, null, 0));
        sched.block(t);
    }
}

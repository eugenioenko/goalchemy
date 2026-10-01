namespace Rt;

/// <summary>core.chan.recv: a pause primitive leaving {value, ok}.</summary>
public static partial class R
{
    public static void chanRecv(GoTask t, Chan ch)
    {
        if (ch == null)
        {
            sched.block(t);
            return;
        }
        var r = tryRecv(ch);
        if ((bool)r[2])
        {
            t.rv = new[] { r[0], r[1] };
            return;
        }
        ch.recvq.Add(new Waiter(t, null, null, 0));
        sched.block(t);
    }
}

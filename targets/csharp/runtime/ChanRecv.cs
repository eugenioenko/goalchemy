namespace Rt;

/// <summary>core.chan.recv: a pause primitive leaving {value, ok}.</summary>
public static partial class R
{
    public static void chanRecv(GoTask t, Chan ch)
    {
        sched.assertTask(t);
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
        var waiter = new Waiter(t, null, null, 0);
        ch.recvq.Add(waiter);
        t.cleanup = () => { ch.recvq.Remove(waiter); waiter.clear(); };
        sched.block(t);
    }
}

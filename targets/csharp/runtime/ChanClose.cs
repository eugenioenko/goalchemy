namespace Rt;

/// <summary>core.chan.close: wakes every receiver and panics blocked senders.</summary>
public static partial class R
{
    public static void chanClose(Chan ch)
    {
        if (ch == null) throw Panics.plainPanic("close of nil channel");
        if (ch.closed) throw Panics.plainPanic("close of closed channel");
        ch.closed = true;
        for (var w = dequeue(ch.recvq); w != null; w = dequeue(ch.recvq)) w.recvDone(ch.zero(), false);
        for (var w = dequeue(ch.sendq); w != null; w = dequeue(ch.sendq)) w.sendDone(true);
    }
}

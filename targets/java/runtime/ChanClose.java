package rt;

/** core.chan.close: close(ch), releasing waiting receivers and senders. */
public final class ChanClose {
    private ChanClose() {}

    public static void chanClose(ChanMake.Chan ch) {
        if (ch == null) throw Panics.plainPanic("close of nil channel");
        if (ch.closed) throw Panics.plainPanic("close of closed channel");
        ch.closed = true;
        for (ChanMake.Waiter w = ChanMake.dequeue(ch.recvq); w != null; w = ChanMake.dequeue(ch.recvq)) w.recvDone(ch.zero.get(), false);
        for (ChanMake.Waiter w = ChanMake.dequeue(ch.sendq); w != null; w = ChanMake.dequeue(ch.sendq)) w.sendDone(true);
    }
}

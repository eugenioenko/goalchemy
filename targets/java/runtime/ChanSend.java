package rt;

/** core.chan.send: ch <- v, a pause primitive. */
public final class ChanSend {
    private ChanSend() {}

    public static void chanSend(TaskSpawn.Task t, ChanMake.Chan ch, Object v) {
        t.rv = new Object[0];
        if (ch == null) {
            TaskSpawn.sched.block(t);
            return;
        }
        if (ch.closed) throw Panics.plainPanic("send on closed channel");
        ChanMake.Waiter r = ChanMake.dequeue(ch.recvq);
        if (r != null) {
            r.recvDone(v, true);
            return;
        }
        if (ch.buf.size() < ch.size) {
            ch.buf.add(v);
            return;
        }
        ChanMake.Waiter waiter=new ChanMake.Waiter(t,v,null,0);
        ch.sendq.add(waiter);
        t.cleanup=()->{ ch.sendq.remove(waiter); waiter.clear(); };
        TaskSpawn.sched.block(t);
    }
}

package rt;

/** core.chan.recv: v, ok := <-ch, a pause primitive; results arrive in
 * t.rv as {value, ok}. */
public final class ChanRecv {
    private ChanRecv() {}

    public static void chanRecv(TaskSpawn.Task t, ChanMake.Chan ch) {
        if (ch == null) {
            TaskSpawn.sched.block(t);
            return;
        }
        Object[] r = ChanMake.tryRecv(ch);
        if ((Boolean) r[2]) {
            t.rv = new Object[] {r[0], r[1]};
            return;
        }
        ch.recvq.add(new ChanMake.Waiter(t, null, null, 0));
        TaskSpawn.sched.block(t);
    }
}

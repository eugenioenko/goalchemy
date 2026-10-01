package rt;

import java.util.ArrayList;

/** core.select: commit exactly one ready case; a pause primitive whose
 * results arrive in t.rv as {index, value, ok}, index -1 for the default.
 * Each case is {channel, isSend, value}. */
public final class Select {
    private Select() {}

    public static Object[] scase(ChanMake.Chan ch, boolean send, Object v) {
        return new Object[] {ch, send, v};
    }

    public static void select(TaskSpawn.Task t, boolean hasDefault, Object[]... cases) {
        ArrayList<Integer> ready = new ArrayList<>();
        for (int i = 0; i < cases.length; i++) {
            ChanMake.Chan ch = (ChanMake.Chan) cases[i][0];
            if (ch == null) continue;
            if ((Boolean) cases[i][1]) {
                if (ch.closed || ChanMake.hasLive(ch.recvq) || ch.buf.size() < ch.size) ready.add(i);
            } else if (!ch.buf.isEmpty() || ChanMake.hasLive(ch.sendq) || ch.closed) {
                ready.add(i);
            }
        }
        if (!ready.isEmpty()) {
            int i = ready.get(TaskSpawn.sched.choose(ready.size()));
            ChanMake.Chan ch = (ChanMake.Chan) cases[i][0];
            if ((Boolean) cases[i][1]) {
                if (ch.closed) throw Panics.plainPanic("send on closed channel");
                ChanMake.Waiter w = ChanMake.dequeue(ch.recvq);
                if (w != null) w.recvDone(cases[i][2], true);
                else ch.buf.add(cases[i][2]);
                t.rv = new Object[] {(long) i, null, false};
                return;
            }
            Object[] r = ChanMake.tryRecv(ch);
            t.rv = new Object[] {(long) i, r[0], r[1]};
            return;
        }
        if (hasDefault) {
            t.rv = new Object[] {-1L, null, false};
            return;
        }
        ChanMake.SelectState st = new ChanMake.SelectState();
        for (int i = 0; i < cases.length; i++) {
            ChanMake.Chan ch = (ChanMake.Chan) cases[i][0];
            if (ch == null) continue;
            boolean send = (Boolean) cases[i][1];
            ChanMake.Waiter w = new ChanMake.Waiter(t, send ? cases[i][2] : null, st, i);
            (send ? ch.sendq : ch.recvq).add(w);
        }
        TaskSpawn.sched.block(t);
        t.cleanup = () -> {
            for (Object[] c : cases) {
                ChanMake.Chan ch = (ChanMake.Chan) c[0];
                if (ch != null) {
                    ch.sendq.removeIf(w -> w.sel == st);
                    ch.recvq.removeIf(w -> w.sel == st);
                }
            }
        };
    }
}

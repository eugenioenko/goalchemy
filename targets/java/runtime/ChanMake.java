package rt;

import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.function.Supplier;

/** core.chan.make: make(chan T, size), and the channel state shared by the
 * channel and select runtime functions. */
public final class ChanMake {
    private ChanMake() {}

    public static final class SelectState {
        boolean done;
    }

    public static final class Waiter {
        final TaskSpawn.Task task;
        final Object val;
        final SelectState sel;
        final int idx;

        Waiter(TaskSpawn.Task task, Object val, SelectState sel, int idx) {
            this.task = task;
            this.val = val;
            this.sel = sel;
            this.idx = idx;
        }

        void recvDone(Object v, boolean ok) {
            if (sel != null) {
                sel.done = true;
                task.rv = new Object[] {(long) idx, v, ok};
            } else {
                task.rv = new Object[] {v, ok};
            }
            TaskSpawn.sched.ready(task);
        }

        void sendDone(boolean closed) {
            if (sel != null) {
                sel.done = true;
                task.rv = new Object[] {(long) idx, null, false};
            } else {
                task.rv = new Object[0];
            }
            if (closed) task.resumePanic = Panics.plainPanic("send on closed channel");
            TaskSpawn.sched.ready(task);
        }
    }

    public static final class Chan {
        final ArrayDeque<Object> buf = new ArrayDeque<>();
        public final int size;
        public boolean closed;
        ArrayList<Waiter> recvq = new ArrayList<>();
        ArrayList<Waiter> sendq = new ArrayList<>();
        final Supplier<Object> zero;

        Chan(int size, Supplier<Object> zero) {
            this.size = size;
            this.zero = zero;
        }

        public ArrayDeque<Object> buffer() {
            return buf;
        }
    }

    static Waiter dequeue(ArrayList<Waiter> q) {
        while (!q.isEmpty()) {
            Waiter w = q.remove(0);
            if (w.sel == null || !w.sel.done) return w;
        }
        return null;
    }

    static boolean hasLive(ArrayList<Waiter> q) {
        for (Waiter w : q) if (w.sel == null || !w.sel.done) return true;
        return false;
    }

    /** Receives without blocking: {value, ok, done}. */
    static Object[] tryRecv(Chan ch) {
        if (!ch.buf.isEmpty()) {
            Object v = ch.buf.poll();
            Waiter w = dequeue(ch.sendq);
            if (w != null) {
                ch.buf.add(w.val);
                w.sendDone(false);
            }
            return new Object[] {v, true, true};
        }
        Waiter w = dequeue(ch.sendq);
        if (w != null) {
            w.sendDone(false);
            return new Object[] {w.val, true, true};
        }
        if (ch.closed) return new Object[] {ch.zero.get(), false, true};
        return new Object[] {null, false, false};
    }

    public static Chan makeChan(long size, Supplier<Object> zero) {
        if (size < 0 || size > (1L << 53)) throw Panics.plainPanic("makechan: size out of range");
        return new Chan((int) size, zero);
    }
}

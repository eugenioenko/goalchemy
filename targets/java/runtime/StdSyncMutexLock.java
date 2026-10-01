package rt;

import java.util.ArrayList;

/** std.sync.mutex.lock: acquire or wait FIFO; Unlock hands the lock over. */
public final class StdSyncMutexLock {
    private StdSyncMutexLock() {}

    public static final class Mutex {
        boolean locked;
        ArrayList<TaskSpawn.Task> waiters = new ArrayList<>();

        public Mutex $clone() {
            Mutex m = new Mutex();
            m.$set(this);
            return m;
        }

        public void $set(Mutex o) {
            locked = o.locked;
            waiters = new ArrayList<>(o.waiters);
        }
    }

    public static void stdSyncMutexLock(TaskSpawn.Task t, Mutex m) {
        t.rv = new Object[0];
        if (!m.locked) {
            m.locked = true;
            return;
        }
        m.waiters.add(t);
        TaskSpawn.sched.block(t);
    }
}

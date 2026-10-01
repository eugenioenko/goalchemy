package rt;

import java.util.ArrayList;

/** std.sync.waitgroup.add: adjust the counter; release waiters at zero. */
public final class StdSyncWaitgroupAdd {
    private StdSyncWaitgroupAdd() {}

    public static final class WaitGroup {
        long n;
        ArrayList<TaskSpawn.Task> waiters = new ArrayList<>();

        public WaitGroup $clone() {
            WaitGroup w = new WaitGroup();
            w.$set(this);
            return w;
        }

        public void $set(WaitGroup o) {
            n = o.n;
            waiters = new ArrayList<>(o.waiters);
        }
    }

    public static void stdSyncWaitgroupAdd(WaitGroup wg, long d) {
        wg.n += d;
        if (wg.n < 0) throw new GoPanic(new Box(Program.STRING_TYPE, "sync: negative WaitGroup counter"));
        if (wg.n == 0) {
            for (TaskSpawn.Task t : wg.waiters) TaskSpawn.sched.ready(t);
            wg.waiters = new ArrayList<>();
        }
    }
}

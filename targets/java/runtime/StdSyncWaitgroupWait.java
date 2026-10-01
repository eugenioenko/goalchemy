package rt;

/** std.sync.waitgroup.wait: return at zero, otherwise wait; a pause primitive. */
public final class StdSyncWaitgroupWait {
    private StdSyncWaitgroupWait() {}

    public static void stdSyncWaitgroupWait(TaskSpawn.Task t, StdSyncWaitgroupAdd.WaitGroup wg) {
        t.rv = new Object[0];
        if (wg.n == 0) return;
        wg.waiters.add(t);
        TaskSpawn.sched.block(t);
    }
}

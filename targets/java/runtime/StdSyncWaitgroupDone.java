package rt;

/** std.sync.waitgroup.done: Add(-1). */
public final class StdSyncWaitgroupDone {
    private StdSyncWaitgroupDone() {}

    public static void stdSyncWaitgroupDone(StdSyncWaitgroupAdd.WaitGroup wg) {
        StdSyncWaitgroupAdd.stdSyncWaitgroupAdd(wg, -1);
    }
}

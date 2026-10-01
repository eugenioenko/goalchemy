package rt;

/** std.sync.mutex.unlock: hand the lock to the first waiter or release it. */
public final class StdSyncMutexUnlock {
    private StdSyncMutexUnlock() {}

    public static void stdSyncMutexUnlock(StdSyncMutexLock.Mutex m) {
        if (!m.locked) TaskSpawn.fatal("sync: unlock of unlocked mutex");
        if (!m.waiters.isEmpty()) {
            TaskSpawn.sched.ready(m.waiters.remove(0));
            return;
        }
        m.locked = false;
    }
}

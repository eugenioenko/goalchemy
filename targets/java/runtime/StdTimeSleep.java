package rt;

/** std.time.sleep: suspend for d nanoseconds of scheduler time; a pause primitive. */
public final class StdTimeSleep {
    private StdTimeSleep() {}

    public static void stdTimeSleep(TaskSpawn.Task t, long d) {
        t.rv = new Object[0];
        if (d <= 0) {
            TaskSpawn.yieldTask(t);
            return;
        }
        TaskSpawn.sched.addTimer(d, t, null);
        TaskSpawn.sched.block(t);
    }
}

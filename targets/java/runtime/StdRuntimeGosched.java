package rt;

/** std.runtime.gosched: requeue the running task; a pause primitive. */
public final class StdRuntimeGosched {
    private StdRuntimeGosched() {}

    public static void stdRuntimeGosched(TaskSpawn.Task t) {
        t.rv = new Object[0];
        TaskSpawn.yieldTask(t);
    }
}

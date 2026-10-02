package rt;

/** std.context.with_timeout: a child cancelled after a scheduler-clock duration. */
public final class StdContextWithTimeout {
    private StdContextWithTimeout() {}

    public static Object[] stdContextWithTimeout(StdContextErr.Context parent, long d) {
        StdContextErr.Context c = StdContextErr.newChild(parent);
        long at=TaskSpawn.deadlineAfter(TaskSpawn.sched.now(),d);
        c.deadline=c.deadline==null ? at : Math.min(c.deadline,at);
        StdContextErr.observe(c);
        if (c.err == null) c.stopTimer=TaskSpawn.sched.addTimerAt(c.deadline, null, () -> StdContextErr.cancel(c, StdContextErr.CONTEXT_DEADLINE_EXCEEDED));
        Fn cancel = a -> {
            StdContextErr.cancel(c, StdContextErr.CONTEXT_CANCELED);
            return null;
        };
        return new Object[] {c, cancel};
    }
}

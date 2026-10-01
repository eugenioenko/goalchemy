package rt;

/** std.context.with_timeout: a child cancelled after a virtual-time duration. */
public final class StdContextWithTimeout {
    private StdContextWithTimeout() {}

    public static Object[] stdContextWithTimeout(StdContextErr.Context parent, long d) {
        StdContextErr.Context c = StdContextErr.newChild(parent);
        if (c.err == null) TaskSpawn.sched.addTimer(d, null, () -> StdContextErr.cancel(c, StdContextErr.CONTEXT_DEADLINE_EXCEEDED));
        Fn cancel = a -> {
            StdContextErr.cancel(c, StdContextErr.CONTEXT_CANCELED);
            return null;
        };
        return new Object[] {c, cancel};
    }
}

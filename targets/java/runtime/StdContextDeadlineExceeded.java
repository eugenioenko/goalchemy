package rt;

/** std.context.deadline_exceeded: the context.DeadlineExceeded sentinel. */
public final class StdContextDeadlineExceeded {
    private StdContextDeadlineExceeded() {}

    public static Box stdContextDeadlineExceeded() {
        return StdContextErr.CONTEXT_DEADLINE_EXCEEDED;
    }
}

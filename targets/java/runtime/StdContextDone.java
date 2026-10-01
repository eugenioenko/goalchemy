package rt;

/** std.context.done: the channel closed on cancellation. */
public final class StdContextDone {
    private StdContextDone() {}

    public static ChanMake.Chan stdContextContextDone(StdContextErr.Context c) {
        return c.done;
    }
}

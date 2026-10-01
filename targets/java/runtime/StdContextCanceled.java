package rt;

/** std.context.canceled: the context.Canceled sentinel. */
public final class StdContextCanceled {
    private StdContextCanceled() {}

    public static Box stdContextCanceled() {
        return StdContextErr.CONTEXT_CANCELED;
    }
}

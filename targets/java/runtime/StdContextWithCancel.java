package rt;

/** std.context.with_cancel: a child cancelled by its cancel function. */
public final class StdContextWithCancel {
    private StdContextWithCancel() {}

    public static Object[] stdContextWithCancel(StdContextErr.Context parent) {
        StdContextErr.Context c = StdContextErr.newChild(parent);
        Fn cancel = a -> {
            StdContextErr.cancel(c, StdContextErr.CONTEXT_CANCELED);
            return null;
        };
        return new Object[] {c, cancel};
    }
}

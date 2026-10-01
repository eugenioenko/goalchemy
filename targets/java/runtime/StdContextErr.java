package rt;

import java.util.ArrayList;

/** std.context.err and the context representation: a cancellation state
 * with a Done channel closed on cancellation; Background's channel is nil. */
public final class StdContextErr {
    private StdContextErr() {}

    public static final class Context {
        final ChanMake.Chan done;
        Box err;
        ArrayList<Context> children = new ArrayList<>();

        Context(ChanMake.Chan done) {
            this.done = done;
        }
    }

    public static final Box CONTEXT_CANCELED = StdErrorsNew.stdErrorsNew("context canceled");
    public static final Box CONTEXT_DEADLINE_EXCEEDED = StdErrorsNew.stdErrorsNew("context deadline exceeded");
    public static final Context BACKGROUND = new Context(null);

    static void cancel(Context c, Box err) {
        if (c.err != null) return;
        c.err = err;
        ChanClose.chanClose(c.done);
        for (Context k : c.children) cancel(k, err);
        c.children = new ArrayList<>();
    }

    static Context newChild(Context parent) {
        Context c = new Context(ChanMake.makeChan(0, () -> null));
        if (parent.err != null) cancel(c, parent.err);
        else if (parent != BACKGROUND) parent.children.add(c);
        return c;
    }

    public static Box stdContextContextErr(Context c) {
        return c.err;
    }
}

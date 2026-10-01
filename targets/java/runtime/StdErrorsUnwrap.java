package rt;

/** std.errors.unwrap: the result of an Unwrap() error method, or nil. */
public final class StdErrorsUnwrap {
    private StdErrorsUnwrap() {}

    public static Box stdErrorsUnwrap(Box err) {
        if (err == null) return null;
        Fn unwrap = err.t.methods.get("Unwrap");
        return unwrap == null ? null : (Box) unwrap.call(err.v);
    }

}

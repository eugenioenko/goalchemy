package rt;

/** std.errors.is: errors.Is over the Unwrap() error chain. */
public final class StdErrorsIs {
    private StdErrorsIs() {}

    public static boolean stdErrorsIs(Box err, Box target) {
        if (err == null || target == null) return err == target;
        boolean comparable = target.t.comparable;
        Box cur = err;
        while (cur != null) {
            if (comparable && cur.t == target.t && cur.t.eq.eq(cur.v, target.v)) return true;
            Fn is = cur.t.methods.get("Is");
            if (is != null && (Boolean) is.call(cur.v, target)) return true;
            Fn unwrap = cur.t.methods.get("Unwrap");
            if (unwrap == null) return false;
            cur = (Box) unwrap.call(cur.v);
        }
        return false;
    }

}

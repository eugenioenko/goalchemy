package rt;

/** A Go function value. Arguments and results are boxed; a function with
 * several results returns an Object[]; one with none returns null. */
public interface Fn {
    Object call(Object... a);

    /** Identity compared by recover; null when unknown. */
    default Object fid() {
        return null;
    }
}

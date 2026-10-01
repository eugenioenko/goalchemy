package rt;

/** A pointer to non-aggregate storage. Pointers to structs and arrays are
 * the stable host objects themselves. */
public interface Ref {
    Object get();

    void set(Object v);
}

package rt;

import java.util.Arrays;

/** A non-nil interface value: dynamic type and value. */
public final class Box {
    public final TypeDesc t;
    public final Object v;

    public Box(TypeDesc t, Object v) {
        this.t = t;
        this.v = v;
    }

    public static Box box(TypeDesc t, Object v) {
        return new Box(t, v);
    }

    public static boolean ifaceEq(Box a, Box b) {
        if (a == null || b == null) return a == b;
        if (a.t != b.t) return false;
        return a.t.eq.eq(a.v, b.v);
    }

    public static Object ifaceKey(Box a) {
        if (a == null) return "nil";
        return Arrays.asList(a.t.id, a.t.key.key(a.v));
    }

    public static String implementsAll(TypeDesc t, String... ids) {
        for (String id : ids) if (!t.methods.containsKey(id)) return id;
        return null;
    }
}

package rt;

import java.util.HashMap;
import java.util.Map;

/** A dynamic type: name, equality, key encoding, and method table. */
public final class TypeDesc {
    public interface Eq {
        boolean eq(Object a, Object b);
    }

    public interface Key {
        Object key(Object a);
    }

    private static int nextId = 1;

    public final int id;
    public final String name;
    public final String kind;
    public final Eq eq;
    public final Key key;
    public final Map<String, Fn> methods;
    public final String basic;
    public final boolean comparable;

    public TypeDesc(String name, String kind, Eq eq, Key key, Map<String, Fn> methods, String basic, boolean comparable) {
        this.id = nextId++;
        this.name = name;
        this.kind = kind;
        this.eq = eq;
        this.key = key;
        this.methods = methods == null ? new HashMap<>() : methods;
        this.basic = basic;
        this.comparable = comparable;
    }

    @SafeVarargs
    public static Map<String, Fn> methods(Object... kv) {
        Map<String, Fn> m = new HashMap<>();
        for (int i = 0; i + 1 < kv.length; i += 2) m.put((String) kv[i], (Fn) kv[i + 1]);
        return m;
    }
}

package rt;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;
import java.util.WeakHashMap;
import java.util.function.BiConsumer;
import java.util.function.Function;

/** Program-level runtime: deferred calls, recover, closures, pointers to
 * fields, and the entry point reporting unrecovered panics as Go does. */
public final class Program {
    private Program() {}

    public static final class Deferred {
        public final Fn f;
        public final Object[] args;
        public final Object fid;
        public final boolean start;

        public Deferred(Fn f, Object[] args, Object fid, boolean start) {
            this.f = f;
            this.args = args;
            this.fid = fid;
            this.start = start;
        }
    }

    /** Per-task recover state. */
    public static class PanicState {
        public GoPanic curPanic;
        public Object deferTarget;
    }

    private static final PanicState MAIN_STATE = new PanicState();

    public interface StateSource {
        PanicState get();
    }

    public static StateSource panicState = () -> MAIN_STATE;

    public static void resetPanicBinding() {
        MAIN_STATE.curPanic=null; MAIN_STATE.deferTarget=null;
        panicState=()->MAIN_STATE;
    }

    public static final TypeDesc PANIC_NIL_ERROR = new TypeDesc("*runtime.PanicNilError", "runtime_error",
            (a, b) -> a == b, a -> a,
            TypeDesc.methods("Error", (Fn) a -> "runtime error: panic called with nil argument", "RuntimeError", (Fn) a -> null),
            null, true);

    public static final TypeDesc STRING_TYPE = new TypeDesc("string", "string", (a, b) -> a.equals(b), a -> a, null, "string", true);

    public static GoPanic goPanic(Box v) {
        return new GoPanic(v == null ? new Box(PANIC_NIL_ERROR, new Object()) : v);
    }

    public static GoPanic catchPanic(Throwable e) {
        if (e instanceof GoPanic g) return g;
        if (e instanceof RuntimeException r) throw r;
        if (e instanceof Error r) throw r;
        throw new RuntimeException(e);
    }

    public static void runDefers(ArrayList<Deferred> ds, GoPanic p) {
        GoPanic panicking = p;
        while (!ds.isEmpty()) {
            Deferred d = ds.remove(ds.size() - 1);
            PanicState st = panicState.get();
            GoPanic savedPanic = st.curPanic;
            Object savedTarget = st.deferTarget;
            st.curPanic = panicking;
            st.deferTarget = d.fid;
            try {
                if (d.f == null) throw Panics.nilDeref();
                d.f.call(d.args);
            } catch (GoPanic np) {
                if (np != panicking && np.prev == null) np.prev = panicking;
                panicking = np;
                continue;
            } finally {
                st.curPanic = savedPanic;
                st.deferTarget = savedTarget;
            }
            if (panicking != null && panicking.recovered) panicking = null;
        }
        if (panicking != null) throw panicking;
    }

    public static Box recover(Object fid) {
        PanicState st = panicState.get();
        GoPanic p = st.curPanic;
        if (p == null || p.recovered || !java.util.Objects.equals(st.deferTarget, fid)) return null;
        p.recovered = true;
        return p.value;
    }

    /** A function value tagged with the identity recover compares against. */
    public static Fn closure(Object fid, Fn f) {
        return new Fn() {
            public Object call(Object... a) {
                return f.call(a);
            }

            public Object fid() {
                return fid;
            }
        };
    }

    public static Fn bound(Object fid, Fn f, Object recv) {
        return closure(fid, a -> {
            Object[] all = new Object[a.length + 1];
            all[0] = recv;
            System.arraycopy(a, 0, all, 1, a.length);
            return f.call(all);
        });
    }

    public static Box ichk(Box x) {
        if (x == null) throw Panics.nilDeref();
        return x;
    }

    public static Fn ibound(Box x, String id) {
        Box b = ichk(x);
        Fn m = b.t.methods.get(id);
        return bound(m.fid(), m, b.v);
    }

    public static Fn fnchk(Fn f) {
        if (f == null) throw Panics.nilDeref();
        return f;
    }

    public static Object fid(Fn f) {
        return f == null ? null : f.fid();
    }

    /** Calls a method from a type's table with the receiver first. */
    public static Object icall(Box x, String id, Object... args) {
        Box b = ichk(x);
        Object[] all = new Object[args.length + 1];
        all[0] = b.v;
        System.arraycopy(args, 0, all, 1, args.length);
        return b.t.methods.get(id).call(all);
    }

    public static GoPanic assertPanic(Box x, String iface, String target, String missing) {
        String msg;
        if (x == null) msg = "interface conversion: " + iface + " is nil, not " + target;
        else if (missing != null) msg = "interface conversion: " + x.t.name + " is not " + target + ": missing method " + missing;
        else msg = "interface conversion: " + iface + " is " + x.t.name + ", not " + target;
        return new GoPanic(new Box(Panics.TYPE_ASSERTION_ERROR, msg));
    }

    private static final Map<Object, HashMap<String, Ref>> refs = new WeakHashMap<>();

    /** Library retirement drops field-pointer graphs after copied host results. */
    public static void clearFieldRefs() { TaskSpawn.sched.assertDriver(); refs.clear(); }

    /** The canonical pointer to a struct field, so pointer equality holds. */
    public static Ref fieldRef(Object o, String k, Function<Object, Object> get, BiConsumer<Object, Object> set) {
        HashMap<String, Ref> m = refs.computeIfAbsent(o, x -> new HashMap<>());
        return m.computeIfAbsent(k, x -> new Ref() {
            public Object get() {
                return get.apply(o);
            }

            public void set(Object v) {
                set.accept(o, v);
            }
        });
    }

    private static String indented(String s) {
        return s.replace("\n", "\n\t");
    }

    public static String formatPanicValue(Box v) {
        if (v == null) return "nil";
        Fn err = v.t.methods.get("Error");
        if (err != null) return indented((String) err.call(v.v));
        Fn str = v.t.methods.get("String");
        if (str != null) return indented((String) str.call(v.v));
        boolean builtin = !v.t.name.contains(".");
        if ("string".equals(v.t.basic)) return builtin ? indented((String) v.v) : v.t.name + "(\"" + indented((String) v.v) + "\")";
        if ("float32".equals(v.t.basic) || "float64".equals(v.t.basic)) {
            String s = Floats.print(((Number)v.v).doubleValue(), "float32".equals(v.t.basic) ? 32 : 64);
            return builtin ? s : v.t.name + "(" + s + ")";
        }
        if ("bool".equals(v.t.basic) || "int".equals(v.t.basic) || "uint".equals(v.t.basic)) {
            String s = v.v instanceof Long l && "uint".equals(v.t.basic) ? Long.toUnsignedString(l) : String.valueOf(v.v);
            return builtin ? s : v.t.name + "(" + s + ")";
        }
        return "(" + v.t.name + ") 0xc000000000";
    }

    public static String formatChain(GoPanic p) {
        String s = "";
        if (p.prev != null) s += formatChain(p.prev) + "\t";
        s += "panic: " + formatPanicValue(p.value);
        if (p.recovered) s += " [recovered]";
        return s + "\n";
    }

    public static void reportPanic(GoPanic p) {
        Out.stderr(formatChain(p));
        System.exit(2);
    }

    /** Runs a body on a thread with a large stack so deep recursion behaves. */
    public static void runLarge(Runnable body) {
        Throwable[] fault = new Throwable[1];
        Thread t = new Thread(null, () -> {
            try {
                body.run();
            } catch (GoPanic p) {
                reportPanic(p);
            } catch (StackOverflowError e) {
                Out.stderr("runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n");
                System.exit(2);
            } catch (Throwable e) {
                fault[0] = e;
            }
        }, "main", 1L << 30);
        t.start();
        try {
            t.join();
        } catch (InterruptedException e) {
            throw new RuntimeException(e);
        }
        if (fault[0] != null) {
            fault[0].printStackTrace();
            System.exit(1);
        }
    }

    public static void main(Runnable entry) {
        runLarge(entry);
        System.out.flush();
        System.exit(0);
    }
}

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.function.Supplier;
import rt.*;

/** Canonical value encoding for the Java harness. Go strings are binary
 * Strings holding one char per byte. */
class Codec {
    Codec() {}

    static String latin1(byte[] b) {
        return new String(b, StandardCharsets.ISO_8859_1);
    }

    static String utf8(String binary) {
        return new String(binary.getBytes(StandardCharsets.ISO_8859_1), StandardCharsets.UTF_8);
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> obj(Object raw) {
        return (Map<String, Object>) raw;
    }

    static boolean isNil(Object raw) {
        return raw instanceof Map<?, ?> m && Boolean.TRUE.equals(m.get("nil"));
    }

    static long decInt(Object raw, String kind) {
        return new BigInteger((String) raw).longValue();
    }

    static boolean decBool(Object raw) {
        return Boolean.TRUE.equals(raw) || "true".equals(raw);
    }

    static String decString(Object raw) {
        Map<String, Object> m = obj(raw);
        if (m.containsKey("str")) return latin1(((String) m.get("str")).getBytes(StandardCharsets.UTF_8));
        String h = (String) m.get("hex");
        byte[] b = new byte[h.length() / 2];
        for (int i = 0; i < b.length; i++) b[i] = (byte) Integer.parseInt(h.substring(2 * i, 2 * i + 2), 16);
        return latin1(b);
    }

    static Box decError(Object raw) {
        if (isNil(raw)) return null;
        return StdErrorsNew.stdErrorsNew(latin1(((String) obj(raw).get("error")).getBytes(StandardCharsets.UTF_8)));
    }

    @SuppressWarnings("unchecked")
    static Slice decSlice(Object raw, Function<Object, Object> dec, Supplier<Object> zero) {
        if (isNil(raw)) return Slice.NIL;
        List<Object> items = (List<Object>) obj(raw).get("slice");
        Object c = obj(raw).get("cap");
        int cap = c == null ? items.size() : Integer.parseInt((String) c);
        Object[] a = new Object[cap];
        for (int i = 0; i < cap; i++) a[i] = i < items.size() ? dec.apply(items.get(i)) : zero.get();
        return new Slice(a, 0, items.size(), cap);
    }

    @SuppressWarnings("unchecked")
    static Slice decBytes(Object raw) {
        if (isNil(raw)) return Slice.BYTE_NIL;
        List<Object> items = (List<Object>) obj(raw).get("slice");
        Object c = obj(raw).get("cap");
        int cap = c == null ? items.size() : Integer.parseInt((String) c);
        byte[] a = new byte[cap];
        for (int i = 0; i < items.size(); i++) a[i] = (byte) decInt(items.get(i), "u8");
        return new Slice(a, 0, items.size(), cap);
    }

    static Slice view(Slice base, Object raw) {
        Map<String, Object> m = obj(raw);
        int lo = m.containsKey("lo") ? Integer.parseInt((String) m.get("lo")) : 0;
        int hi = m.containsKey("hi") ? Integer.parseInt((String) m.get("hi")) : base.l;
        int mx = m.containsKey("max") ? Integer.parseInt((String) m.get("max")) : base.c;
        return new Slice(base.a, base.o + lo, hi - lo, mx - lo, base.bytes);
    }

    @SuppressWarnings("unchecked")
    static GoMap decMap(Object raw, Function<Object, Object> dk, Function<Object, Object> dv) {
        if (isNil(raw)) return null;
        GoMap m = new GoMap(GoMap::identityKey);
        for (Object e : (List<Object>) obj(raw).get("map")) MapStore.mapSet(m, dk.apply(obj(e).get("key")), dv.apply(obj(e).get("value")));
        return m;
    }

    @SuppressWarnings("unchecked")
    static ChanMake.Chan decChan(Object raw, Function<Object, Object> dec, Supplier<Object> zero) {
        if (isNil(raw)) return null;
        Object c = obj(raw).get("cap");
        ChanMake.Chan ch = ChanMake.makeChan(c == null ? 0 : Long.parseLong((String) c), zero);
        for (Object v : (List<Object>) obj(raw).get("chan")) ch.buffer().add(dec.apply(v));
        if (Boolean.TRUE.equals(obj(raw).get("closed"))) ch.closed = true;
        return ch;
    }

    static Object encInt(Object v, String kind) {
        long x = (Long) v;
        return kind.equals("u64") ? Long.toUnsignedString(x) : Long.toString(x);
    }

    static Object encBool(Object v) {
        return (Boolean) v ? "true" : "false";
    }

    static Object encString(Object v) {
        String s = (String) v;
        StringBuilder b = new StringBuilder();
        for (int i = 0; i < s.length(); i++) b.append(String.format("%02x", (int) s.charAt(i)));
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("hex", b.toString());
        return m;
    }

    static Object encError(Object v) {
        Map<String, Object> m = new LinkedHashMap<>();
        if (v == null) {
            m.put("nil", true);
            return m;
        }
        Box b = (Box) v;
        m.put("error", utf8((String) b.t.methods.get("Error").call(b.v)));
        return m;
    }

    static Object encSlice(Object v, Function<Object, Object> enc) {
        Slice s = (Slice) v;
        Map<String, Object> m = new LinkedHashMap<>();
        if (s.a == null) {
            m.put("nil", true);
            return m;
        }
        List<Object> items = new ArrayList<>();
        for (int i = 0; i < s.l; i++) items.add(enc.apply(s.get(i)));
        m.put("slice", items);
        m.put("cap", Integer.toString(s.c));
        return m;
    }

    static Object encArray(Object v, Function<Object, Object> enc) {
        List<Object> items = new ArrayList<>();
        if (v instanceof byte[] a) {
            for (byte x : a) items.add(enc.apply((long) (x & 255)));
        } else {
            for (Object x : (Object[]) v) items.add(enc.apply(x));
        }
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("array", items);
        return m;
    }

    static Object encMap(Object v, Function<Object, Object> ek, Function<Object, Object> ev) {
        Map<String, Object> m = new LinkedHashMap<>();
        if (v == null) {
            m.put("nil", true);
            return m;
        }
        GoMap.Iter it = MapIterate.mapIter((GoMap) v);
        List<Object> items = new ArrayList<>();
        while (MapIterate.mapNext(it)) {
            Map<String, Object> e = new LinkedHashMap<>();
            e.put("key", ek.apply(it.k));
            e.put("value", ev.apply(it.v));
            items.add(e);
        }
        m.put("map", items);
        return m;
    }

    static Object encChan(Object v, Function<Object, Object> enc) {
        Map<String, Object> m = new LinkedHashMap<>();
        if (v == null) {
            m.put("nil", true);
            return m;
        }
        ChanMake.Chan ch = (ChanMake.Chan) v;
        List<Object> items = new ArrayList<>();
        for (Object x : ch.buffer()) items.add(enc.apply(x));
        m.put("chan", items);
        m.put("cap", Integer.toString(ch.size));
        m.put("closed", ch.closed);
        return m;
    }

    static Object encZero(Object v) {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("zero", true);
        return m;
    }

    /** Reports whether a spawned task ran before the parent yielded. */
    static final class SpawnCheck extends TaskSpawn.Frame {
        boolean ran;
        boolean before;

        public void step(TaskSpawn.Task t) {
            if (pc == 0) {
                TaskSpawn.spawn(TaskSpawn.sync(() -> {
                    ran = true;
                    return new Object[0];
                }));
                before = ran;
                pc = 1;
                StdRuntimeGosched.stdRuntimeGosched(t);
                return;
            }
            if (!ran) throw new IllegalStateException("spawned task did not run after the parent yielded");
            TaskSpawn.ret(t, this);
        }

        public Object[] results() {
            return new Object[] {before};
        }
    }

    static TaskSpawn.Frame newSpawnCheck() {
        return new SpawnCheck();
    }

    static void harnessSelect2(TaskSpawn.Task t, ChanMake.Chan a, ChanMake.Chan b, boolean dflt) {
        Select.select(t, dflt, Select.scase(a, false, null), Select.scase(b, false, null));
    }
    static Object encKey(Object value){if(value!=null)throw new IllegalStateException("key fixture must be nil");return java.util.Map.of("nil",true);}
    static Object[] host(TaskSpawn.Primitive primitive) {
        return TaskSpawn.driveLibrary(()->TaskSpawn.nativeFrame(primitive),rv->rv.clone(),()->{},w->{});
    }
}

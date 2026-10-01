package rt;

import java.util.ArrayList;
import java.util.HashMap;

/** Go maps: insertion-ordered entries with snapshot iteration. */
public final class GoMap {
    public interface KeyOf {
        Object key(Object k);
    }

    public static final class Entry {
        public final Object k;
        public Object v;
        public boolean live = true;

        Entry(Object k, Object v) {
            this.k = k;
            this.v = v;
        }
    }

    public final HashMap<Object, Entry> index = new HashMap<>();
    public ArrayList<Entry> entries = new ArrayList<>();
    public final KeyOf keyOf;

    public GoMap(KeyOf keyOf) {
        this.keyOf = keyOf;
    }

    public void add(Object key, Object k, Object v) {
        Entry e = new Entry(k, v);
        index.put(key, e);
        entries.add(e);
    }

    public void compact() {
        if (entries.size() > 32 && entries.size() > 2 * index.size()) {
            ArrayList<Entry> live = new ArrayList<>();
            for (Entry e : entries) if (e.live) live.add(e);
            entries = live;
        }
    }

    public static Object identityKey(Object k) {
        return k;
    }

    public static final class Iter {
        public final Entry[] entries;
        public int i;
        public Object k;
        public Object v;

        public Iter(Entry[] entries) {
            this.entries = entries;
        }
    }
}

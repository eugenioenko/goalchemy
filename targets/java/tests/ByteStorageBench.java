import java.util.Locale;
import rt.*;

/** Fresh-JVM diagnostic; mandatory tests assert representation, not heap noise. */
public final class ByteStorageBench {
    private static Slice[] retained;

    private static void gc() throws InterruptedException {
        System.gc();
        Thread.sleep(100);
        System.gc();
        Thread.sleep(100);
    }

    private static long used() {
        Runtime r = Runtime.getRuntime();
        return r.totalMemory() - r.freeMemory();
    }

    public static void main(String[] args) throws Exception {
        boolean bytes = args[0].equals("native");
        if (!bytes && !args[0].equals("boxed")) throw new IllegalArgumentException("native|boxed MiB");
        int n = Math.multiplyExact(Integer.parseInt(args[1]), 1024 * 1024);
        // Warm both runtime paths independently of the measured retained payload.
        for (int pass = 0; pass < 2000; pass++) {
            Slice s = SliceMake.makeSlice(256, 256, () -> 0L, bytes);
            SliceCopy.copy(s, s, null);
            SliceAppend.appendSlice(s, s, null);
        }
        gc();
        long before = used();
        Slice src = SliceMake.makeSlice(n, n, () -> 0L, bytes);
        Slice dst = SliceMake.makeSlice(n, n, () -> 0L, bytes);
        if (bytes) {
            byte[] a = (byte[]) src.a;
            for (int i = 0; i < n; i++) a[i] = (byte) i;
        } else {
            Object[] a = (Object[]) src.a;
            for (int i = 0; i < n; i++) a[i] = (long) (i & 255);
        }
        long start = System.nanoTime();
        for (int i = 0; i < 16; i++) SliceCopy.copy(dst, src, null);
        long copyTime = System.nanoTime() - start;
        start = System.nanoTime();
        Slice appended = SliceAppend.appendSlice(src, src, null);
        long appendTime = System.nanoTime() - start;
        retained = new Slice[] {src, dst, appended};
        gc();
        long delta = used() - before;
        long backing = 0;
        if (bytes) {
            for (Slice s : retained) backing += ((byte[]) s.a).length;
            if (backing != 4L * n) throw new AssertionError("retained byte backing size");
        }
        for (Slice s : retained) {
            if (!s.get(0).equals(0L) || !s.get(s.l - 1).equals(255L)) throw new AssertionError("payload reachability");
        }
        System.out.printf(Locale.ROOT, "mode=%s input=%d retained_backing_bytes=%s heap_delta=%d copy16_ms=%.3f append_ms=%.3f%n",
                args[0], n, bytes ? Long.toString(backing) : "boxed", delta, copyTime / 1e6, appendTime / 1e6);
    }
}

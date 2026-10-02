import java.lang.reflect.Method;
import java.util.Arrays;
import rt.*;

/** Independent storage assertions, run by the default Go contract suite. */
public final class ByteStorageTest {
    private static void check(boolean ok, String label) {
        if (!ok) throw new AssertionError(label);
    }

    private static byte[] backing(Slice s) {
        check(s.bytes && s.a instanceof byte[], "native byte[] backing");
        byte[] a = (byte[]) s.a;
        check(a.length >= s.o + s.c, "physical capacity");
        return a;
    }

    private static void panics(Runnable fn) {
        try { fn.run(); } catch (GoPanic expected) { return; }
        throw new AssertionError("expected source panic");
    }

    private static Slice bytes(int len, int cap) {
        return SliceMake.makeSlice(len, cap, () -> { throw new AssertionError("byte make boxed a zero"); }, true);
    }

    public static void main(String[] args) throws Exception {
        Slice nil = Slice.BYTE_NIL;
        check(nil.a == null && nil.bytes, "typed nil");
        check(SliceSlice.reslice(nil, 0L, 0L, 0L, false) == nil, "nil view hint");
        check(SliceAppend.appendBytes(nil, new byte[0]) == nil, "zero argument append");
        check(SliceAppend.appendSlice(nil, nil, null) == nil, "nil spread hint");
        Slice empty = bytes(0, 0);
        check(backing(empty).length == 0, "allocated empty");
        check(SliceAppend.appendSlice(nil, empty, null) == nil, "empty spread preserves nil");
        Slice s = bytes(2, 4);
        check(Arrays.equals(backing(s), new byte[4]), "make zeroes full capacity");
        SliceStore.bset(s, 0, 255);
        SliceStore.bsetu(s, 1, 128);
        check(SliceIndex.bget(s, 0) == 255L && SliceIndex.bgetu(s, 1) == 128L, "unsigned primitive reads");
        check(SliceIndex.sget(s, 0).equals(255L), "generic scalar boundary remains long");
        SliceStore.bset(s, 0, 511);
        check(SliceIndex.bget(s, 0) == 255L, "byte store wraps");
        Slice grown = SliceAppend.appendBytes(s, new byte[] {3, 4, 5});
        check(grown.a != s.a && backing(grown).length == 8, "growth detaches with prescribed capacity");
        check(Arrays.equals(backing(grown), new byte[] {-1, -128, 3, 4, 5, 0, 0, 0}), "growth zeroes spare capacity");
        Slice inPlace = SliceAppend.appendBytes(s, new byte[] {9});
        check(inPlace.a == s.a && inPlace.l == 3, "in capacity append shares");
        Slice full = SliceSlice.reslice(inPlace, 1L, 2L, 3L, false);
        check(full.o == 1 && full.l == 1 && full.c == 2 && full.a == s.a, "full view");
        SliceClear.clearSlice(full, () -> { throw new AssertionError("byte clear boxed a zero"); });
        check(Arrays.equals(backing(s), new byte[] {-1, 0, 9, 0}), "clear only selected view");
        for (boolean right : new boolean[] {false, true}) {
            Slice b = StringToBytes.toBytes("\1\2\3\4\5\6");
            Slice dst = SliceSlice.reslice(b, right ? 1L : 0L, right ? 6L : 5L, null, false);
            Slice src = SliceSlice.reslice(b, right ? 0L : 1L, right ? 5L : 6L, null, false);
            check(SliceCopy.copy(dst, src, v -> { throw new AssertionError("byte copy boxed"); }) == 5, "overlap count");
            check(Arrays.equals(backing(b), right ? new byte[] {1, 1, 2, 3, 4, 5} : new byte[] {2, 3, 4, 5, 6, 6}), "overlap copy both directions");
        }
        Slice b = StringToBytes.toBytes("\1\2\3\4\5\6");
        Slice a = SliceAppend.appendSlice(SliceSlice.reslice(b, 0L, 2L, null, false), SliceSlice.reslice(b, 1L, 5L, null, false), null);
        check(a.a == b.a && Arrays.equals(backing(b), new byte[] {1, 2, 2, 3, 4, 5}), "overlap append right");
        b = StringToBytes.toBytes("\1\2\3\4\5\6");
        a = SliceAppend.appendSlice(SliceSlice.reslice(b, 0L, 0L, null, false), SliceSlice.reslice(b, 1L, 5L, null, false), null);
        check(a.a == b.a && Arrays.equals(backing(b), new byte[] {2, 3, 4, 5, 5, 6}), "overlap append left");
        Slice detached = SliceAppend.appendSlice(SliceSlice.reslice(b, 1L, 3L, 3L, false), b, null);
        check(detached.a != b.a && backing(detached).length == 8, "spread growth native");
        check(Arrays.equals(backing(detached), new byte[] {3, 4, 2, 3, 4, 5, 5, 6}), "spread growth retains old source");
        String binary = "\0\377\200\300\257A";
        Slice binaryBytes = StringToBytes.toBytes(binary);
        check(Arrays.equals(backing(binaryBytes), new byte[] {0, -1, -128, -64, -81, 65}), "arbitrary string bytes");
        String saved = StringFromBytes.fromBytes(binaryBytes);
        SliceStore.bset(binaryBytes, 1, 1);
        check(saved.equals(binary), "string copy independent");
        Slice second = StringToBytes.toBytes(saved);
        check(second.a != binaryBytes.a, "bytes conversion independent");
        check(backing(StringToBytes.toBytes("")).length == 0 && StringFromBytes.fromBytes(nil).isEmpty(), "empty conversions");
        Slice appendedString = SliceAppend.appendString(nil, binary);
        check(Arrays.equals(backing(appendedString), backing(second)), "string append native");
        check(SliceCopy.copyString(binaryBytes, binary) == 6 && Arrays.equals(backing(binaryBytes), backing(second)), "string copy native");
        byte[] array = SliceToArray.sliceToByteArray(second, 4);
        check(Arrays.equals(array, new byte[] {0, -1, -128, -64}) && array != second.a, "slice to array value copy");
        Slice view = SliceSlice.sliceArray(array, 1L, 3L, 4L, false);
        SliceStore.bset(view, 0, 12);
        check(array[1] == 12 && view.a == array, "array view shares native backing");
        check(SliceToArray.sliceToByteArray(nil, 0).length == 0, "nil to zero array");
        panics(() -> SliceIndex.bget(nil, 0));
        panics(() -> SliceStore.bset(nil, 0, 1));
        panics(() -> SliceIndex.bgetu(second, -1));
        panics(() -> SliceStore.bsetu(second, -1, 1));
        panics(() -> SliceSlice.reslice(second, 0L, 7L, null, false));
        panics(() -> SliceMake.makeSlice(-1, 0, () -> 0L, true));
        panics(() -> SliceMake.makeSlice(2, 1, () -> 0L, true));
        panics(() -> SliceToArray.sliceToByteArray(second, 7));
        byte[] small = {42};
        Slice impossible = new Slice(small, 0, Integer.MAX_VALUE, Integer.MAX_VALUE);
        try {
            SliceAppend.appendBytes(impossible, new byte[] {1});
            throw new AssertionError("overflow accepted");
        } catch (IllegalStateException expected) {
            check(small[0] == 42, "overflow before mutation");
        }
        try {
            SliceMake.makeSlice(0, Long.MAX_VALUE, () -> 0L, true);
            throw new AssertionError("huge make accepted");
        } catch (GoPanic expected) { }
        Slice ints = SliceMake.makeSlice(2, 3, () -> 0L);
        check(!ints.bytes && ints.a instanceof Object[], "generic storage retained");
        SliceStore.sset(ints, 0, 1000L);
        SliceStore.sset(ints, 1, -1L);
        Slice generic = SliceAppend.appendSlice(Slice.NIL, ints, null);
        check(generic.a instanceof Object[] && generic.get(0).equals(1000L) && generic.get(1).equals(-1L), "generic append retained");
        SliceCopy.copy(ints, generic, null);
        SliceClear.clearSlice(ints, () -> 0L);
        check(ints.get(0).equals(0L) && ints.get(1).equals(0L), "generic copy and clear");
        check(SliceIndex.sgetu(generic, 1).equals(-1L), "generic unsigned index");

        // Inspect compiler-generated primitive array helpers, independently of
        // emitted-text checks and the language fixture's observable values.
        int helpers = 0;
        for (Method zero : Main.class.getDeclaredMethods()) {
            if (!zero.getName().startsWith("zero_") || zero.getReturnType() != byte[].class) continue;
            byte[] orig = (byte[]) zero.invoke(null);
            if (orig.length == 0) continue;
            String id = zero.getName().substring(5);
            orig[0] = (byte) 255;
            byte[] copy = (byte[]) Main.class.getDeclaredMethod("clone_" + id, byte[].class).invoke(null, orig);
            check(copy != orig && Arrays.equals(copy, orig), "array clone native and detached");
            Method key = Main.class.getDeclaredMethod("key_" + id, byte[].class);
            Object savedKey = key.invoke(null, orig);
            check(savedKey.equals(key.invoke(null, copy)), "byte array keys compare by values");
            Method eq = Main.class.getDeclaredMethod("eq_" + id, byte[].class, byte[].class);
            check((Boolean) eq.invoke(null, orig, copy), "byte array equality");
            orig[0] = 3;
            check(!savedKey.equals(key.invoke(null, orig)), "array key snapshot independent");
            Main.class.getDeclaredMethod("set_" + id, byte[].class, byte[].class).invoke(null, orig, copy);
            check(orig[0] == (byte) 255 && savedKey.equals(key.invoke(null, orig)), "array assignment preserves storage");
            helpers++;
        }
        check(helpers >= 1, "generated byte[] helpers exercised");
        System.out.println("Java native byte storage passed");
    }
}

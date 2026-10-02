using System;
using System.Linq;
using System.Reflection;
using Rt;

/** Independent storage assertions, run by the default Go contract suite. */
public static class ByteStorageTest {
    static bool Same(byte[] a, byte[] b) => a.SequenceEqual(b);

    static void check(bool ok, string label) {
        if (!ok) throw new Exception(label);
    }

    static byte[] backing(Slice s) {
        check(s.bytes && s.a is byte[], "native byte[] backing");
        byte[] a = (byte[]) s.a;
        check(a.Length >= s.o + s.c, "physical capacity");
        return a;
    }

    static void panics(Action fn) {
        try { fn(); } catch (GoPanic) { return; }
        throw new Exception("expected source panic");
    }

    static void faults(Action fn) {
        try { fn(); } catch (Fault) { return; }
        throw new Exception("expected host fault");
    }

    static Slice bytes(int len, int cap) {
        return R.makeSlice(len, cap, () => { throw new Exception("byte make boxed a zero"); }, true);
    }

    public static void Main(string[] args)  {
        Slice nil = Slice.BYTE_NIL;
        check(nil.a == null && nil.bytes, "typed nil");
        check(R.reslice(nil, 0L, 0L, 0L, false) == nil, "nil view hint");
        check(R.appendBytes(nil, new byte[0]) == nil, "zero argument append");
        check(R.appendSlice(nil, nil, null) == nil, "nil spread hint");
        Slice empty = bytes(0, 0);
        check(backing(empty).Length == 0, "allocated empty");
        check(R.appendBytes(empty, Array.Empty<byte>()) == empty, "zero append preserves allocated empty");
        Slice fromNil = R.appendBytes(nil, new byte[] {255, 128});
        check(Same(backing(fromNil), new byte[] {255, 128}), "append to nil allocates native storage");
        check(R.appendSlice(nil, empty, null) == nil, "empty spread preserves nil");
        Slice s = bytes(2, 4);
        check(Same(backing(s), new byte[4]), "make zeroes full capacity");
        R.bset(s, 0, 255);
        R.bsetu(s, 1, 128);
        check(R.bget(s, 0) == 255L && R.bgetu(s, 1) == 128L, "unsigned primitive reads");
        check(R.sget(s, 0) is long && R.sget(s, 0).Equals(255L), "generic scalar boundary remains long");
        R.sset(s, 0, 255L);
        check(R.bget(s, 0) == 255L, "generic store accepts boxed long");
        R.bset(s, 0, 511);
        check(R.bget(s, 0) == 255L, "byte store wraps");
        Slice grown = R.appendBytes(s, new byte[] {3, 4, 5});
        check(grown.a != s.a && backing(grown).Length == 8, "growth detaches with prescribed capacity");
        check(Same(backing(grown), new byte[] {255, 128, 3, 4, 5, 0, 0, 0}), "growth zeroes spare capacity");
        Slice inPlace = R.appendBytes(s, new byte[] {9});
        check(inPlace.a == s.a && inPlace.l == 3, "in capacity append shares");
        Slice full = R.reslice(inPlace, 1L, 2L, 3L, false);
        check(full.o == 1 && full.l == 1 && full.c == 2 && full.a == s.a, "full view");
        R.clearSlice(full, () => { throw new Exception("byte clear boxed a zero"); });
        check(Same(backing(s), new byte[] {255, 0, 9, 0}), "clear only selected view");
        foreach (bool right in new bool[] {false, true}) {
            Slice overlapBuffer = R.toBytes("\u0001\u0002\u0003\u0004\u0005\u0006");
            Slice dst = R.reslice(overlapBuffer, right ? 1L : 0L, right ? 6L : 5L, null, false);
            Slice src = R.reslice(overlapBuffer, right ? 0L : 1L, right ? 5L : 6L, null, false);
            check(R.copy(dst, src, v => { throw new Exception("byte copy boxed"); }) == 5, "overlap count");
            check(Same(backing(overlapBuffer), right ? new byte[] {1, 1, 2, 3, 4, 5} : new byte[] {2, 3, 4, 5, 6, 6}), "overlap copy both directions");
        }
        Slice b = R.toBytes("\u0001\u0002\u0003\u0004\u0005\u0006");
        Slice a = R.appendSlice(R.reslice(b, 0L, 2L, null, false), R.reslice(b, 1L, 5L, null, false), null);
        check(a.a == b.a && Same(backing(b), new byte[] {1, 2, 2, 3, 4, 5}), "overlap append right");
        b = R.toBytes("\u0001\u0002\u0003\u0004\u0005\u0006");
        a = R.appendSlice(R.reslice(b, 0L, 0L, null, false), R.reslice(b, 1L, 5L, null, false), null);
        check(a.a == b.a && Same(backing(b), new byte[] {2, 3, 4, 5, 5, 6}), "overlap append left");
        Slice detached = R.appendSlice(R.reslice(b, 1L, 3L, 3L, false), b, v => { throw new Exception("byte spread append boxed"); });
        check(detached.a != b.a && backing(detached).Length == 8, "spread growth native");
        check(Same(backing(detached), new byte[] {3, 4, 2, 3, 4, 5, 5, 6}), "spread growth retains old source");
        string binary = "\0\u00ff\u0080\u00c0\u00afA";
        Slice binaryBytes = R.toBytes(binary);
        check(Same(backing(binaryBytes), new byte[] {0, 255, 128, 192, 175, 65}), "arbitrary string bytes");
        string saved = R.fromBytes(binaryBytes);
        R.bset(binaryBytes, 1, 1);
        check(saved.Equals(binary), "string copy independent");
        Slice second = R.toBytes(saved);
        check(second.a != binaryBytes.a, "bytes conversion independent");
        check(backing(R.toBytes("")).Length == 0 && R.fromBytes(nil) == "", "empty conversions");
        Slice appendedString = R.appendString(nil, binary);
        check(Same(backing(appendedString), backing(second)), "string append native");
        check(R.copyString(binaryBytes, binary) == 6 && Same(backing(binaryBytes), backing(second)), "string copy native");
        byte[] array = R.sliceToByteArray(second, 4);
        check(Same(array, new byte[] {0, 255, 128, 192}) && array != second.a, "slice to array value copy");
        Slice view = R.sliceArray(array, 1L, 3L, 4L, false);
        R.bset(view, 0, 12);
        check(array[1] == 12 && view.a == array, "array view shares native backing");
        check(R.sliceToByteArray(nil, 0).Length == 0, "nil to zero array");
        panics(() => R.bget(nil, 0));
        panics(() => R.bset(nil, 0, 1));
        panics(() => R.bgetu(second, -1));
        panics(() => R.bsetu(second, -1, 1));
        panics(() => R.reslice(second, 0L, 7L, null, false));
        panics(() => R.makeSlice(-1, 0, () => 0L, true));
        panics(() => R.makeSlice(2, 1, () => 0L, true));
        panics(() => R.sliceToByteArray(second, 7));
        byte[] small = {42};
        Slice impossible = new Slice(small, 0, int.MaxValue, int.MaxValue);
        faults(() => R.appendBytes(impossible, new byte[] {1}));
        faults(() => R.appendSlice(impossible, fromNil, null));
        faults(() => R.appendString(impossible, "a"));
        faults(() => R.makeSlice(0, int.MaxValue, () => 0L, true));
        check(small[0] == 42, "host-limit rejection before mutation");
        try {
            R.makeSlice(0, long.MaxValue, () => 0L, true);
            throw new Exception("huge make accepted");
        } catch (GoPanic) { }
        Slice ints = R.makeSlice(2, 3, () => 0L);
        check(!ints.bytes && ints.a is object[], "generic storage retained");
        R.sset(ints, 0, 1000L);
        R.sset(ints, 1, -1L);
        Slice generic = R.appendSlice(Slice.NIL, ints, null);
        check(generic.a is object[] && generic.Get(0).Equals(1000L) && generic.Get(1).Equals(-1L), "generic append retained");
        R.copy(ints, generic, null);
        R.clearSlice(ints, () => 0L);
        check(ints.Get(0).Equals(0L) && ints.Get(1).Equals(0L), "generic copy and clear");
        check(R.sgetu(generic, 1).Equals(-1L), "generic unsigned index");

        // Inspect compiler-generated primitive array helpers, independently of
        // emitted-text checks and the language fixture's observable values.
        int helpers = 0;
        var flags = BindingFlags.Static | BindingFlags.NonPublic;
        foreach (var zero in typeof(GoProgram).GetMethods(flags)) {
            if (!zero.Name.StartsWith("zero_") || zero.ReturnType != typeof(byte[])) continue;
            byte[] orig = (byte[])zero.Invoke(null, null);
            if (orig.Length == 0) continue;
            string id = zero.Name.Substring(5);
            orig[0] = 255;
            byte[] copy = (byte[])typeof(GoProgram).GetMethod("clone_" + id, flags).Invoke(null, new object[] {orig});
            check(copy != orig && Same(copy, orig), "array clone native and detached");
            var key = typeof(GoProgram).GetMethod("key_" + id, flags);
            object savedKey = key.Invoke(null, new object[] {orig});
            check(savedKey.Equals(key.Invoke(null, new object[] {copy})), "byte array keys compare by values");
            var eq = typeof(GoProgram).GetMethod("eq_" + id, flags);
            check((bool)eq.Invoke(null, new object[] {orig, copy}), "byte array equality");
            orig[0] = 3;
            check(!savedKey.Equals(key.Invoke(null, new object[] {orig})), "array key snapshot independent");
            typeof(GoProgram).GetMethod("set_" + id, flags).Invoke(null, new object[] {orig, copy});
            check(orig[0] == 255 && savedKey.Equals(key.Invoke(null, new object[] {orig})), "array assignment preserves storage");
            helpers++;
        }
        check(helpers >= 1, "generated byte[] helpers exercised");
        Console.WriteLine("C# native byte storage passed");
    }
}

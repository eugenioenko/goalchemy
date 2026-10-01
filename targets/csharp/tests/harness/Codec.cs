using System.Numerics;
using System.Text;
using System.Text.Json.Nodes;
using Rt;

/// <summary>Canonical value encoding for the C# harness. Go strings hold one
/// char per byte.</summary>
static partial class Harness
{
    internal static string Latin1(byte[] b) => Encoding.Latin1.GetString(b);

    internal static string Utf8Text(string binary) => Encoding.UTF8.GetString(Encoding.Latin1.GetBytes(binary));

    static bool isNil(JsonNode raw) => raw is JsonObject o && o["nil"] is JsonValue v && v.GetValue<bool>();

    static string str(JsonNode n) => n.GetValue<string>();

    static long decInt(JsonNode raw, string kind) => (long)(ulong)(BigInteger.Parse(str(raw)) & ulong.MaxValue);

    static bool decBool(JsonNode raw) => raw is JsonValue v && (v.TryGetValue<bool>(out var b) ? b : v.GetValue<string>() == "true");

    static string decString(JsonNode raw)
    {
        if (raw["str"] is JsonNode s) return Latin1(Encoding.UTF8.GetBytes(str(s)));
        return Latin1(Convert.FromHexString(str(raw["hex"])));
    }

    static Box decError(JsonNode raw)
    {
        if (isNil(raw)) return null;
        return R.stdErrorsNew(Latin1(Encoding.UTF8.GetBytes(str(raw["error"]))));
    }

    static Slice decSlice(JsonNode raw, Func<JsonNode, object> dec, Func<object> zero)
    {
        if (isNil(raw)) return Slice.NIL;
        var items = raw["slice"].AsArray();
        int cap = raw["cap"] is JsonNode c ? int.Parse(str(c)) : items.Count;
        var a = new object[cap];
        for (int i = 0; i < cap; i++) a[i] = i < items.Count ? dec(items[i]) : zero();
        return new Slice(a, 0, items.Count, cap);
    }

    static Slice view(Slice b, JsonNode raw)
    {
        int lo = raw["lo"] is JsonNode l ? int.Parse(str(l)) : 0;
        int hi = raw["hi"] is JsonNode h ? int.Parse(str(h)) : b.l;
        int mx = raw["max"] is JsonNode m ? int.Parse(str(m)) : b.c;
        return new Slice(b.a, b.o + lo, hi - lo, mx - lo);
    }

    static GoMap decMap(JsonNode raw, Func<JsonNode, object> dk, Func<JsonNode, object> dv)
    {
        if (isNil(raw)) return null;
        var m = new GoMap(GoMap.identityKey);
        foreach (var e in raw["map"].AsArray()) R.mapSet(m, dk(e["key"]), dv(e["value"]));
        return m;
    }

    static Chan decChan(JsonNode raw, Func<JsonNode, object> dec, Func<object> zero)
    {
        if (isNil(raw)) return null;
        var ch = R.makeChan(raw["cap"] is JsonNode c ? long.Parse(str(c)) : 0, zero);
        foreach (var v in raw["chan"].AsArray()) ch.buf.Enqueue(dec(v));
        if (raw["closed"] is JsonValue cl && cl.GetValue<bool>()) ch.closed = true;
        return ch;
    }

    static JsonNode encInt(object v, string kind)
    {
        long x = (long)v;
        return JsonValue.Create(kind == "u64" ? ((ulong)x).ToString() : x.ToString());
    }

    static JsonNode encBool(object v) => JsonValue.Create((bool)v ? "true" : "false");

    static JsonNode encString(object v) => new JsonObject { ["hex"] = Convert.ToHexString(Encoding.Latin1.GetBytes((string)v)).ToLowerInvariant() };

    static JsonNode encError(object v)
    {
        if (v == null) return new JsonObject { ["nil"] = true };
        var b = (Box)v;
        return new JsonObject { ["error"] = Utf8Text((string)b.t.Methods["Error"].Call(b.v)) };
    }

    static JsonNode encSlice(object v, Func<object, JsonNode> enc)
    {
        var s = (Slice)v;
        if (s.a == null) return new JsonObject { ["nil"] = true };
        var items = new JsonArray();
        for (int i = 0; i < s.l; i++) items.Add(enc(s.a[s.o + i]));
        return new JsonObject { ["slice"] = items, ["cap"] = s.c.ToString() };
    }

    static JsonNode encArray(object v, Func<object, JsonNode> enc)
    {
        var items = new JsonArray();
        foreach (var x in (object[])v) items.Add(enc(x));
        return new JsonObject { ["array"] = items };
    }

    static JsonNode encMap(object v, Func<object, JsonNode> ek, Func<object, JsonNode> ev)
    {
        if (v == null) return new JsonObject { ["nil"] = true };
        var it = R.mapIter((GoMap)v);
        var items = new JsonArray();
        while (R.mapNext(it)) items.Add(new JsonObject { ["key"] = ek(it.k), ["value"] = ev(it.v) });
        return new JsonObject { ["map"] = items };
    }

    static JsonNode encChan(object v, Func<object, JsonNode> enc)
    {
        if (v == null) return new JsonObject { ["nil"] = true };
        var ch = (Chan)v;
        var items = new JsonArray();
        foreach (var x in ch.buf) items.Add(enc(x));
        return new JsonObject { ["chan"] = items, ["cap"] = ch.size.ToString(), ["closed"] = ch.closed };
    }

    static JsonNode encZero(object v) => new JsonObject { ["zero"] = true };

    /// <summary>Reports whether a spawned task ran before the parent yielded.</summary>
    sealed class SpawnCheck : Frame
    {
        bool ran;
        bool before;

        public override void step(GoTask t)
        {
            if (pc == 0)
            {
                R.spawn(R.sync(() =>
                {
                    ran = true;
                    return Array.Empty<object>();
                }));
                before = ran;
                pc = 1;
                R.stdRuntimeGosched(t);
                return;
            }
            if (!ran) throw new InvalidOperationException("spawned task did not run after the parent yielded");
            R.ret(t, this);
        }

        public override object[] results() => new object[] { before };
    }

    static Frame newSpawnCheck() => new SpawnCheck();

    static void harnessSelect2(GoTask t, Chan a, Chan b, bool dflt) =>
        R.select(t, dflt, R.scase(a, false, null), R.scase(b, false, null));
}

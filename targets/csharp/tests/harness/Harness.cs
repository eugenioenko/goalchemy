using System.Text;
using System.Text.Json.Nodes;
using Rt;

/// <summary>Serves runtime conformance requests for the C# target over JSON
/// Lines on standard input and output.</summary>
static partial class Harness
{
    const int PROTOCOL = 1;

    internal sealed class H
    {
        readonly JsonObject let;
        internal readonly JsonObject after = new();

        internal H(JsonObject let) => this.let = let;

        internal JsonNode Let(string n) => let[n];

        internal void After(string n, JsonNode v) => after[n] = v;
    }

    static JsonObject Serve(JsonObject req)
    {
        var resp = new JsonObject { ["v"] = PROTOCOL, ["id"] = req["id"]?.DeepClone() };
        if (req["v"] is not JsonValue v || v.GetValue<int>() != PROTOCOL)
        {
            resp["status"] = "harness_failure";
            resp["error"] = "unsupported protocol version";
            return resp;
        }
        var name = req["case"]?.GetValue<string>();
        if (name == null || !CASES.TryGetValue(name, out var fn))
        {
            resp["status"] = "harness_failure";
            resp["error"] = "unknown case " + name;
            return resp;
        }
        var h = new H(req["let"] as JsonObject ?? new JsonObject());
        try
        {
            R.resetScheduler();
            var results = fn(h);
            resp["status"] = "returned";
            resp["results"] = results;
            resp["after"] = h.after;
        }
        catch (Blocked)
        {
            resp["status"] = "blocked";
        }
        catch (GoPanic p)
        {
            resp["status"] = "panic";
            resp["panic"] = Utf8Text(Program.formatPanicValue(p.value));
        }
        catch (Exception e)
        {
            resp["status"] = "harness_failure";
            resp["error"] = e.GetType().Name + ": " + e.Message;
        }
        return resp;
    }

    static void Run()
    {
        var stdin = new StreamReader(Console.OpenStandardInput(), new UTF8Encoding(false));
        var stdout = new StreamWriter(Console.OpenStandardOutput(), new UTF8Encoding(false));
        string line;
        while ((line = stdin.ReadLine()) != null)
        {
            line = line.Trim();
            if (line.Length == 0) continue;
            JsonObject resp;
            try
            {
                resp = Serve(JsonNode.Parse(line).AsObject());
            }
            catch (Exception e) when (e is System.Text.Json.JsonException || e is InvalidOperationException)
            {
                resp = new JsonObject { ["v"] = PROTOCOL, ["status"] = "harness_failure", ["error"] = e.Message };
            }
            stdout.Write(resp.ToJsonString() + "\n");
            stdout.Flush();
        }
    }

    static void Main()
    {
        var t = new System.Threading.Thread(Run, 1 << 29);
        t.Start();
        t.Join();
    }
}

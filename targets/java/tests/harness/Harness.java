import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import rt.*;

/** Serves runtime conformance requests for the Java target over JSON Lines
 * on standard input and output. */
public final class Harness {
    static final int PROTOCOL = 1;

    static final class H {
        final Map<String, Object> let;
        final Map<String, Object> after = new LinkedHashMap<>();

        H(Map<String, Object> let) {
            this.let = let;
        }

        Object let(String n) {
            return let.get(n);
        }

        void after(String n, Object v) {
            after.put(n, v);
        }
    }

    interface Case {
        List<Object> run(H h);
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> serve(Map<String, Object> req) {
        Map<String, Object> resp = new LinkedHashMap<>();
        resp.put("v", PROTOCOL);
        resp.put("id", req.get("id"));
        if (!"1".equals(req.get("v"))) {
            resp.put("status", "harness_failure");
            resp.put("error", "unsupported protocol version");
            return resp;
        }
        Case fn = HarnessGen.CASES.get((String) req.get("case"));
        if (fn == null) {
            resp.put("status", "harness_failure");
            resp.put("error", "unknown case " + req.get("case"));
            return resp;
        }
        Object let = req.get("let");
        H h = new H(let == null ? Map.of() : (Map<String, Object>) let);
        try {
            TaskSpawn.resetScheduler();
            List<Object> results = fn.run(h);
            resp.put("status", "returned");
            resp.put("results", results);
            resp.put("after", h.after);
        } catch (TaskSpawn.Blocked e) {
            resp.put("status", "blocked");
        } catch (GoPanic p) {
            resp.put("status", "panic");
            resp.put("panic", Codec.utf8(Program.formatPanicValue(p.value)));
        } catch (Throwable e) {
            resp.put("status", "harness_failure");
            resp.put("error", e.toString());
        }
        return resp;
    }

    @SuppressWarnings("unchecked")
    public static void main(String[] args) throws Exception {
        BufferedReader in = new BufferedReader(new InputStreamReader(System.in, StandardCharsets.UTF_8));
        PrintStream out = new PrintStream(System.out, false, StandardCharsets.UTF_8);
        String line;
        while ((line = in.readLine()) != null) {
            line = line.trim();
            if (line.isEmpty()) continue;
            Map<String, Object> resp;
            try {
                resp = serve((Map<String, Object>) Json.parse(line));
            } catch (IllegalArgumentException | ClassCastException e) {
                resp = new LinkedHashMap<>();
                resp.put("v", PROTOCOL);
                resp.put("status", "harness_failure");
                resp.put("error", e.toString());
            }
            out.print(Json.write(resp) + "\n");
            out.flush();
        }
    }
}

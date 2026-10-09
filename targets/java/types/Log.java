package rt;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Host log sink for std/log/slog. Levels follow log/slog: -4 debug, 0 info,
 * 4 warn, 8 error. Without a handler, records at warn and above are written
 * to standard error.
 */
public final class Log {
    private Log() {}

    /** One record; attrs are key/value pairs with group names joined to keys by ".". */
    public record Record(int level, long unixNano, String message, List<Map.Entry<String, String>> attrs, String text) {
        public Instant time() { return Instant.ofEpochSecond(Math.floorDiv(unixNano, 1000000000L), Math.floorMod(unixNano, 1000000000L)); }
    }

    @FunctionalInterface
    public interface Handler { void handle(Record record); }

    private static final class Sink {
        final Handler handler; final long level;
        Sink(Handler handler, long level) { this.handler = handler; this.level = level; }
    }

    private static volatile Sink sink = new Sink(null, 4);

    /** Routes records at level and above to handler; null restores standard error. */
    public static void setHandler(Handler handler, int level) { sink = new Sink(handler, level); }

    public static void setHandler(Handler handler) { setHandler(handler, 4); }

    /** A handler that forwards records to the System.Logger named "goalchemy". */
    public static Handler systemLogger() { return systemLogger(System.getLogger("goalchemy")); }

    public static Handler systemLogger(System.Logger logger) {
        return r -> {
            System.Logger.Level level = r.level() < 0 ? System.Logger.Level.DEBUG : r.level() < 4 ? System.Logger.Level.INFO
                    : r.level() < 8 ? System.Logger.Level.WARNING : System.Logger.Level.ERROR;
            if (logger.isLoggable(level)) logger.log(level, r.text());
        };
    }

    public static boolean enabled(long level) { return level >= sink.level; }

    private static String text(String binary) {
        return new String(binary.getBytes(StandardCharsets.ISO_8859_1), StandardCharsets.UTF_8);
    }

    public static void emit(long level, long unixNano, String message, Slice attrs, String line) {
        Handler handler = sink.handler;
        if (handler == null) { Out.stderr(line + "\n"); return; }
        List<Map.Entry<String, String>> pairs = new ArrayList<>();
        for (int i = 0; i + 1 < attrs.l; i += 2) pairs.add(Map.entry(text((String) attrs.get(i)), text((String) attrs.get(i + 1))));
        int clamped = (int) Math.max(Integer.MIN_VALUE, Math.min(Integer.MAX_VALUE, level));
        try { handler.handle(new Record(clamped, unixNano, text(message), List.copyOf(pairs), text(line))); } catch (RuntimeException e) {}
    }
}

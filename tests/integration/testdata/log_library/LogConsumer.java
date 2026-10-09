package consumer;

import io.goalchemy.generated.Generated;
import io.goalchemy.runtime.Library;
import io.goalchemy.runtime.Log;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.TimeUnit;

public final class LogConsumer {
    static void check(boolean value, String label) {
        if (!value) throw new AssertionError(label);
    }
    static <T> T get(Library.Operation<T> operation) throws Exception {
        return operation.completion().toCompletableFuture().get(20, TimeUnit.SECONDS);
    }
    public static void main(String[] args) throws Exception {
        List<Log.Record> got = new ArrayList<>();
        Log.setHandler(got::add, -4);
        check(get(Generated.Work(3L, null)) == 6L, "result");
        check(got.size() == 4, "record count " + got.size());
        check(got.get(0).level() == -4 && got.get(0).text().equals("level=DEBUG msg=start sdk=probe n=3"), "debug record");
        check(got.get(1).text().equals("level=INFO msg=info sdk=probe unicode=\"héllo wörld\"") && got.get(1).attrs().get(1).getValue().equals("héllo wörld"), "utf8 decoding");
        check(got.get(2).level() == 4 && got.get(2).message().equals("retry") && got.get(2).attrs().get(1).getKey().equals("kas.url"), "group attrs");
        check(got.get(3).text().equals("level=ERROR msg=failed err=boom") && Duration.between(got.get(3).time(), Instant.now()).abs().toSeconds() < 60, "error record");
        got.clear();
        Log.setHandler(got::add, 4);
        get(Generated.Work(1L, null));
        check(got.size() == 2 && got.get(0).message().equals("retry") && got.get(1).level() == 8, "host level");
        List<String> logged = new ArrayList<>();
        System.Logger logger = new System.Logger() {
            public String getName() { return "goalchemy"; }
            public boolean isLoggable(Level level) { return level.getSeverity() >= Level.INFO.getSeverity(); }
            public void log(Level level, java.util.ResourceBundle bundle, String msg, Throwable thrown) { logged.add(level + " " + msg); }
            public void log(Level level, java.util.ResourceBundle bundle, String format, Object... params) { logged.add(level + " " + format); }
        };
        Log.setHandler(Log.systemLogger(logger), -4);
        get(Generated.Work(1L, null));
        check(logged.equals(List.of("INFO level=INFO msg=info sdk=probe unicode=\"héllo wörld\"", "WARNING level=WARN msg=retry sdk=probe kas.url=https://kas kas.attempt=2", "ERROR level=ERROR msg=failed err=boom")), "System.Logger " + logged);
        Log.setHandler(r -> { throw new IllegalStateException("sink"); }, -4);
        check(get(Generated.Work(1L, null)) == 2L, "sink errors are discarded");
        Log.setHandler(null);
        System.out.println("PASS log library");
    }
}

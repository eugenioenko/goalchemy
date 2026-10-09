package rt;
public final class LibLogEnabled {
 private LibLogEnabled() {}
 public static boolean libLogEnabled(long level){return Log.enabled(level);}
}

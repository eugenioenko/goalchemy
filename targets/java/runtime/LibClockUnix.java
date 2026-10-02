package rt;
public final class LibClockUnix {
 private LibClockUnix() {}
 public static long libClockUnix(){return java.time.Instant.now().getEpochSecond();}
}

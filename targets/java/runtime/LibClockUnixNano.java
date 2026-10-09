package rt;
public final class LibClockUnixNano {
 private LibClockUnixNano() {}
 public static long libClockUnixNano(){java.time.Instant now=java.time.Instant.now();return Math.addExact(Math.multiplyExact(now.getEpochSecond(),1000000000L),now.getNano());}
}

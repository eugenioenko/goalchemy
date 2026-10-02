package rt;

public final class LibCryptoRandom {
 private LibCryptoRandom() {}
 public static void libCryptoRandom(TaskSpawn.Task t, long a0) {
  Crypto.execute(t,"random",new Native.Key[]{},new Object[]{a0},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}

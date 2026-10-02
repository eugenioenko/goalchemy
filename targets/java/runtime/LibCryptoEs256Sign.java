package rt;

public final class LibCryptoEs256Sign {
 private LibCryptoEs256Sign() {}
 public static void libCryptoEs256Sign(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"es_sign",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}

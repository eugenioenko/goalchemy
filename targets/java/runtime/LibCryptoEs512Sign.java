package rt;

public final class LibCryptoEs512Sign {
 private LibCryptoEs512Sign() {}
 public static void libCryptoEs512Sign(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"es512_sign",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}

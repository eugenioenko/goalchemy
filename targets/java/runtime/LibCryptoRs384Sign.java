package rt;

public final class LibCryptoRs384Sign {
 private LibCryptoRs384Sign() {}
 public static void libCryptoRs384Sign(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rs384_sign",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
